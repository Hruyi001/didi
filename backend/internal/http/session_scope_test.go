package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"didi/backend/internal/domain"
	"didi/backend/internal/services"
	"didi/backend/internal/store"
)

func TestPassengerOrdersOnlyReturnAuthenticatedPassengerOrders(t *testing.T) {
	s := store.NewMemoryStore()
	passengerA := s.SeedPassenger("13800000001")
	passengerB := s.SeedPassenger("13800000002")
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	s.CreateOrder(domain.RideOrder{PassengerID: passengerA.ID, Pickup: domain.Point{Name: "A"}, Dropoff: domain.Point{Name: "B"}})
	s.CreateOrder(domain.RideOrder{PassengerID: passengerB.ID, Pickup: domain.Point{Name: "C"}, Dropoff: domain.Point{Name: "D"}})
	passengerToken := loginTokenForPhone(t, router, domain.RolePassenger, "13800000001")

	req := httptest.NewRequest(http.MethodGet, "/api/passenger/orders", nil)
	req.Header.Set("Authorization", "Bearer "+passengerToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, passengerA.ID) {
		t.Fatalf("expected own passenger id in body, got %s", body)
	}
	if strings.Contains(body, passengerB.ID) {
		t.Fatalf("did not expect other passenger id in body, got %s", body)
	}
}

func TestDriverProfileUsesAuthenticatedDriverPhone(t *testing.T) {
	s := store.NewMemoryStore()
	s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京B54321")
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)
	driverToken := loginTokenForPhone(t, router, domain.RoleDriver, "13900000002")

	req := httptest.NewRequest(http.MethodGet, "/api/driver/profile", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, driverB.ID) {
		t.Fatalf("expected authenticated driver in body, got %s", body)
	}
	if strings.Contains(body, "13900000001") {
		t.Fatalf("did not expect first driver profile in body, got %s", body)
	}
}

func TestPassengerCannotOperateAnotherPassengersOrder(t *testing.T) {
	s := store.NewMemoryStore()
	s.SeedPassenger("13800000001")
	passengerB := s.SeedPassenger("13800000002")
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	if _, err := s.UpdateDriverLocation(driver.ID, domain.DriverLocation{DriverID: driver.ID, Lng: 116.4, Lat: 39.9, SpeedKPH: 40}); err != nil {
		t.Fatalf("update driver location: %v", err)
	}
	assignedOrder := s.CreateOrder(domain.RideOrder{PassengerID: passengerB.ID, DriverID: driver.ID, Status: domain.OrderWaitingPickup, Pickup: domain.Point{Name: "A"}, Dropoff: domain.Point{Name: "B"}})
	paymentOrder := s.CreateOrder(domain.RideOrder{PassengerID: passengerB.ID, Status: domain.OrderWaitingPayment, Pickup: domain.Point{Name: "C"}, Dropoff: domain.Point{Name: "D"}})
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)
	passengerToken := loginTokenForPhone(t, router, domain.RolePassenger, "13800000001")

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "cancel", method: http.MethodPost, path: "/api/passenger/orders/" + assignedOrder.ID + "/cancel"},
		{name: "pay", method: http.MethodPost, path: "/api/passenger/orders/" + paymentOrder.ID + "/pay"},
		{name: "review", method: http.MethodPost, path: "/api/passenger/orders/" + assignedOrder.ID + "/review", body: `{"score":5,"content":"good"}`},
		{name: "driver-location", method: http.MethodGet, path: "/api/passenger/orders/" + assignedOrder.ID + "/driver-location"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader *bytes.Reader
			if tt.body == "" {
				bodyReader = bytes.NewReader(nil)
			} else {
				bodyReader = bytes.NewReader([]byte(tt.body))
			}
			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			req.Header.Set("Authorization", "Bearer "+passengerToken)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d body=%s", res.Code, res.Body.String())
			}
		})
	}
}

func TestDriverLocationUsesAuthenticatedDriverIdentity(t *testing.T) {
	s := store.NewMemoryStore()
	driverA := s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京B54321")
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)
	driverToken := loginTokenForPhone(t, router, domain.RoleDriver, "13900000001")

	payload := bytes.NewReader([]byte(`{"driverId":"` + driverB.ID + `","lng":116.397,"lat":39.908,"speedKph":35}`))
	req := httptest.NewRequest(http.MethodPost, "/api/driver/location", payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+driverToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Body.String(), driverB.ID) {
		t.Fatalf("did not expect spoofed driver id in body, got %s", res.Body.String())
	}
	if !strings.Contains(res.Body.String(), driverA.ID) {
		t.Fatalf("expected authenticated driver id in body, got %s", res.Body.String())
	}
	if _, ok := s.GetDriverLocation(driverB.ID); ok {
		t.Fatal("did not expect spoofed driver location to be written")
	}
	if _, ok := s.GetDriverLocation(driverA.ID); !ok {
		t.Fatal("expected authenticated driver location to be written")
	}
}

func TestDriverCannotAdvanceAnotherDriversOrder(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driverA := s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京B54321")
	driverC := s.SeedApprovedDriver("13900000003", "京C54321")
	s.SeedApprovedDriver("13900000004", "京D54321")
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	if _, err := s.SetDriverWorkStatus(driverA.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver A online: %v", err)
	}
	if _, err := s.SetDriverWorkStatus(driverB.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver B online: %v", err)
	}
	if _, err := s.SetDriverWorkStatus(driverC.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver C online: %v", err)
	}

	waitingPickup := s.CreateOrder(domain.RideOrder{PassengerID: passenger.ID, Pickup: domain.Point{Name: "A"}, Dropoff: domain.Point{Name: "B"}})
	if _, err := app.Dispatch.Dispatch(waitingPickup.ID); err != nil {
		t.Fatalf("dispatch waiting pickup order: %v", err)
	}
	if _, err := app.Dispatch.Accept(waitingPickup.ID, driverA.ID); err != nil {
		t.Fatalf("accept waiting pickup order: %v", err)
	}
	driverArrived := s.CreateOrder(domain.RideOrder{PassengerID: passenger.ID, Pickup: domain.Point{Name: "C"}, Dropoff: domain.Point{Name: "D"}})
	if _, err := app.Dispatch.Dispatch(driverArrived.ID); err != nil {
		t.Fatalf("dispatch arrived order: %v", err)
	}
	if _, err := app.Dispatch.Accept(driverArrived.ID, driverB.ID); err != nil {
		t.Fatalf("accept arrived order: %v", err)
	}
	if _, err := s.UpdateOrderStatus(driverArrived.ID, domain.OrderDriverArrived); err != nil {
		t.Fatalf("move order to arrived: %v", err)
	}
	inProgress := s.CreateOrder(domain.RideOrder{PassengerID: passenger.ID, Pickup: domain.Point{Name: "E"}, Dropoff: domain.Point{Name: "F"}})
	if _, err := app.Dispatch.Dispatch(inProgress.ID); err != nil {
		t.Fatalf("dispatch in-progress order: %v", err)
	}
	if _, err := app.Dispatch.Accept(inProgress.ID, driverC.ID); err != nil {
		t.Fatalf("accept in-progress order: %v", err)
	}
	if _, err := s.UpdateOrderStatus(inProgress.ID, domain.OrderDriverArrived); err != nil {
		t.Fatalf("move order to arrived for in-progress flow: %v", err)
	}
	if _, err := s.UpdateOrderStatus(inProgress.ID, domain.OrderInProgress); err != nil {
		t.Fatalf("move order to in-progress: %v", err)
	}
	driverToken := loginTokenForPhone(t, router, domain.RoleDriver, "13900000004")

	tests := []struct {
		name string
		path string
	}{
		{name: "arrive", path: "/api/driver/orders/" + waitingPickup.ID + "/arrive"},
		{name: "start", path: "/api/driver/orders/" + driverArrived.ID + "/start"},
		{name: "end", path: "/api/driver/orders/" + inProgress.ID + "/end"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+driverToken)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d body=%s", res.Code, res.Body.String())
			}
		})
	}
}

func TestDriverCannotAcceptOrRejectUnassignedDispatchOffer(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driverA := s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京B54321")
	if _, err := s.SetDriverWorkStatus(driverA.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver A online: %v", err)
	}
	if _, err := s.SetDriverWorkStatus(driverB.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver B online: %v", err)
	}
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	acceptOrder := app.Orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})
	acceptAttempt, err := app.Dispatch.Dispatch(acceptOrder.ID)
	if err != nil {
		t.Fatalf("dispatch accept order: %v", err)
	}
	if acceptAttempt.DriverID != driverA.ID && acceptAttempt.DriverID != driverB.ID {
		t.Fatalf("expected first offer to known driver, got %s", acceptAttempt.DriverID)
	}

	rejectOrder := app.Orders.CreateRide(passenger.ID, domain.Point{Name: "C"}, domain.Point{Name: "D"})
	rejectAttempt, err := app.Dispatch.Dispatch(rejectOrder.ID)
	if err != nil {
		t.Fatalf("dispatch reject order: %v", err)
	}
	if rejectAttempt.DriverID != driverA.ID && rejectAttempt.DriverID != driverB.ID {
		t.Fatalf("expected first offer to known driver, got %s", rejectAttempt.DriverID)
	}

	tests := []struct {
		name       string
		path       string
		offeredID  string
		roguePhone string
	}{
		{name: "accept", path: "/api/driver/orders/" + acceptOrder.ID + "/accept", offeredID: acceptAttempt.DriverID},
		{name: "reject", path: "/api/driver/orders/" + rejectOrder.ID + "/reject", offeredID: rejectAttempt.DriverID},
	}

	for i := range tests {
		if tests[i].offeredID == driverA.ID {
			tests[i].roguePhone = "13900000002"
		} else {
			tests[i].roguePhone = "13900000001"
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			driverToken := loginTokenForPhone(t, router, domain.RoleDriver, tt.roguePhone)
			req := httptest.NewRequest(http.MethodPost, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+driverToken)
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != http.StatusForbidden {
				t.Fatalf("expected 403, got %d body=%s", res.Code, res.Body.String())
			}
		})
	}

	updatedAcceptOrder, ok := s.GetOrder(acceptOrder.ID)
	if !ok {
		t.Fatal("expected accept order to exist")
	}
	if updatedAcceptOrder.DriverID != "" {
		t.Fatalf("did not expect order to be assigned, got driver %s", updatedAcceptOrder.DriverID)
	}
	acceptAttempts := s.ListDispatchAttempts(acceptOrder.ID)
	if len(acceptAttempts) != 1 {
		t.Fatalf("expected 1 accept-order attempt, got %d", len(acceptAttempts))
	}

	rejectAttempts := s.ListDispatchAttempts(rejectOrder.ID)
	if len(rejectAttempts) != 1 {
		t.Fatalf("expected 1 reject-order attempt, got %d", len(rejectAttempts))
	}
	if rejectAttempts[0].Status != domain.DispatchOffered {
		t.Fatalf("expected original offer to remain, got %s", rejectAttempts[0].Status)
	}
}

func loginTokenForPhone(t *testing.T, router http.Handler, role domain.AccountRole, phone string) string {
	t.Helper()
	body := map[string]any{"phone": phone, "code": "123456", "role": role}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	var envelope struct {
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal login response: %v", err)
	}
	if envelope.Data.AccessToken == "" {
		t.Fatal("expected access token")
	}
	return envelope.Data.AccessToken
}
