package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"didi/backend/internal/domain"
	"didi/backend/internal/services"
	"didi/backend/internal/store"
)

func TestDriverOrdersUsesDriverScope(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driverA := s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京A54321")
	if _, err := s.SetDriverWorkStatus(driverA.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver A online: %v", err)
	}
	if _, err := s.SetDriverWorkStatus(driverB.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver B online: %v", err)
	}
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	order := app.Orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})
	if _, err := app.Dispatch.Dispatch(order.ID); err != nil {
		t.Fatalf("dispatch order: %v", err)
	}
	driverToken := loginTokenForPhone(t, router, domain.RoleDriver, "13900000001")

	req := httptest.NewRequest(http.MethodGet, "/api/driver/orders", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, order.ID) {
		t.Fatalf("expected offered order in body, got %s", body)
	}
}

func TestDriverOrdersHideDispatchingOrdersOfferedToAnotherDriver(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driverA := s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京A54321")
	if _, err := s.SetDriverWorkStatus(driverA.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver A online: %v", err)
	}
	if _, err := s.SetDriverWorkStatus(driverB.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver B online: %v", err)
	}
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	order := app.Orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})
	attempt, err := app.Dispatch.Dispatch(order.ID)
	if err != nil {
		t.Fatalf("dispatch order: %v", err)
	}
	if attempt.DriverID != driverA.ID {
		t.Fatalf("expected first offer to driver A, got %s", attempt.DriverID)
	}
	driverBToken := loginTokenForPhone(t, router, domain.RoleDriver, "13900000002")

	req := httptest.NewRequest(http.MethodGet, "/api/driver/orders", nil)
	req.Header.Set("Authorization", "Bearer "+driverBToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if strings.Contains(body, order.ID) {
		t.Fatalf("did not expect other driver's dispatching order in body, got %s", body)
	}
}
