package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"didi/backend/internal/domain"
	"didi/backend/internal/services"
	"didi/backend/internal/store"
)

func TestCreateOrderPublishesRealtimeEvent(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	if _, err := s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver online: %v", err)
	}
	hub := NewEventHub()
	subscriber := hub.Subscribe()
	defer hub.Unsubscribe(subscriber)

	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: hub}
	router := NewRouter(app)

	body := map[string]any{
		"passengerId": passenger.ID,
		"pickup":      map[string]any{"name": "上车点", "lng": 116.397, "lat": 39.908},
		"dropoff":     map[string]any{"name": "下车点", "lng": 116.407, "lat": 39.918},
	}
	payload, _ := json.Marshal(body)
	passengerToken := loginTokenForPhone(t, router, domain.RolePassenger, "13800000001")
	req := httptest.NewRequest(http.MethodPost, "/api/passenger/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+passengerToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	select {
	case event := <-subscriber:
		if event.Type != "order.updated" {
			t.Fatalf("expected order.updated, got %s", event.Type)
		}
		if event.OrderID == "" {
			t.Fatal("expected event to include order id")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected realtime event after order creation")
	}
}

func TestAcceptOrderPublishesWaitingPickupEvent(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	if _, err := s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver online: %v", err)
	}
	hub := NewEventHub()
	subscriber := hub.Subscribe()
	defer hub.Unsubscribe(subscriber)
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: hub}
	router := NewRouter(app)

	order := app.Orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})
	if _, err := app.Dispatch.Dispatch(order.ID); err != nil {
		t.Fatalf("dispatch order: %v", err)
	}
	driverToken := loginTokenForPhone(t, router, domain.RoleDriver, "13900000001")

	req := httptest.NewRequest(http.MethodPost, "/api/driver/orders/"+order.ID+"/accept", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	select {
	case event := <-subscriber:
		if event.Status != string(domain.OrderWaitingPickup) {
			t.Fatalf("expected WAITING_PICKUP, got %s", event.Status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected realtime event after accept")
	}
}

func TestRejectOrderKeepsDispatchingWhenAnotherDriverExists(t *testing.T) {
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
	firstAttempt, err := app.Dispatch.Dispatch(order.ID)
	if err != nil {
		t.Fatalf("dispatch order: %v", err)
	}
	if firstAttempt.DriverID != driverA.ID {
		t.Fatalf("expected first attempt to driver A, got %s", firstAttempt.DriverID)
	}
	driverToken := loginTokenForPhone(t, router, domain.RoleDriver, "13900000001")

	req := httptest.NewRequest(http.MethodPost, "/api/driver/orders/"+order.ID+"/reject", nil)
	req.Header.Set("Authorization", "Bearer "+driverToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}

	updatedOrder, ok := s.GetOrder(order.ID)
	if !ok {
		t.Fatal("expected order to exist")
	}
	if updatedOrder.Status != domain.OrderDispatching {
		t.Fatalf("expected DISPATCHING, got %s", updatedOrder.Status)
	}

	attempts := s.ListDispatchAttempts(order.ID)
	if len(attempts) != 3 {
		t.Fatalf("expected 3 attempts, got %d", len(attempts))
	}
	if attempts[2].DriverID != driverB.ID {
		t.Fatalf("expected redispatch to driver B, got %s", attempts[2].DriverID)
	}
	if attempts[2].Status != domain.DispatchOffered {
		t.Fatalf("expected offered attempt, got %s", attempts[2].Status)
	}
}

func TestRejectOrderPublishesRedispatchEventForNextDriver(t *testing.T) {
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
	hub := NewEventHub()
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: hub}
	router := NewRouter(app)

	order := app.Orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})
	firstAttempt, err := app.Dispatch.Dispatch(order.ID)
	if err != nil {
		t.Fatalf("dispatch order: %v", err)
	}
	if firstAttempt.DriverID != driverA.ID {
		t.Fatalf("expected first attempt to driver A, got %s", firstAttempt.DriverID)
	}
	driverBSubscriber := hub.SubscribeFiltered(realtimeEventFilter(s, services.Session{Role: domain.RoleDriver, Phone: "13900000002"}))
	defer hub.Unsubscribe(driverBSubscriber)
	driverAToken := loginTokenForPhone(t, router, domain.RoleDriver, "13900000001")

	req := httptest.NewRequest(http.MethodPost, "/api/driver/orders/"+order.ID+"/reject", nil)
	req.Header.Set("Authorization", "Bearer "+driverAToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}

	select {
	case event := <-driverBSubscriber:
		if event.Type != "order.updated" {
			t.Fatalf("expected order.updated, got %s", event.Type)
		}
		if event.OrderID != order.ID {
			t.Fatalf("expected redispatched order id %s, got %s", order.ID, event.OrderID)
		}
		if event.Status != string(domain.OrderDispatching) {
			t.Fatalf("expected DISPATCHING status, got %s", event.Status)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected redispatched driver to receive realtime event")
	}
}

func TestRealtimeEventFilteringByIdentity(t *testing.T) {
	s := store.NewMemoryStore()
	passengerA := s.SeedPassenger("13800000001")
	passengerB := s.SeedPassenger("13800000002")
	driverA := s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京B54321")
	if _, err := s.SetDriverWorkStatus(driverA.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver A online: %v", err)
	}
	if _, err := s.SetDriverWorkStatus(driverB.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver B online: %v", err)
	}
	hub := NewEventHub()
	dispatch := services.NewDispatchService(s)

	passengerOrder := s.CreateOrder(domain.RideOrder{PassengerID: passengerA.ID, Pickup: domain.Point{Name: "A"}, Dropoff: domain.Point{Name: "B"}})
	if _, err := dispatch.Dispatch(passengerOrder.ID); err != nil {
		t.Fatalf("dispatch passenger order: %v", err)
	}
	if _, err := dispatch.Accept(passengerOrder.ID, driverA.ID); err != nil {
		t.Fatalf("accept passenger order: %v", err)
	}
	otherPassengerOrder := s.CreateOrder(domain.RideOrder{PassengerID: passengerB.ID, Pickup: domain.Point{Name: "C"}, Dropoff: domain.Point{Name: "D"}})
	if _, err := dispatch.Dispatch(otherPassengerOrder.ID); err != nil {
		t.Fatalf("dispatch other passenger order: %v", err)
	}
	if _, err := dispatch.Accept(otherPassengerOrder.ID, driverB.ID); err != nil {
		t.Fatalf("accept other passenger order: %v", err)
	}

	passengerSubscriber := hub.SubscribeFiltered(realtimeEventFilter(s, services.Session{Role: domain.RolePassenger, Phone: "13800000001"}))
	defer hub.Unsubscribe(passengerSubscriber)
	driverSubscriber := hub.SubscribeFiltered(realtimeEventFilter(s, services.Session{Role: domain.RoleDriver, Phone: "13900000001"}))
	defer hub.Unsubscribe(driverSubscriber)

	hub.Publish(RealtimeEvent{Type: "order.updated", OrderID: passengerOrder.ID, Status: string(domain.OrderWaitingPickup)})
	if event := <-passengerSubscriber; event.OrderID != passengerOrder.ID {
		t.Fatalf("expected passenger event for own order, got %+v", event)
	}
	if event := <-driverSubscriber; event.OrderID != passengerOrder.ID {
		t.Fatalf("expected driver event for assigned order, got %+v", event)
	}

	hub.Publish(RealtimeEvent{Type: "driver.location", DriverID: driverA.ID, Lng: 116.4, Lat: 39.9, SpeedKPH: 30})
	if event := <-passengerSubscriber; event.DriverID != driverA.ID {
		t.Fatalf("expected passenger location for assigned driver, got %+v", event)
	}
	if event := <-driverSubscriber; event.DriverID != driverA.ID {
		t.Fatalf("expected driver location for own driver, got %+v", event)
	}

	hub.Publish(RealtimeEvent{Type: "order.updated", OrderID: otherPassengerOrder.ID, Status: string(domain.OrderWaitingPickup)})
	select {
	case event := <-passengerSubscriber:
		t.Fatalf("did not expect passenger to receive other order event, got %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
	select {
	case event := <-driverSubscriber:
		t.Fatalf("did not expect driver to receive other driver order event, got %+v", event)
	case <-time.After(100 * time.Millisecond):
	}

	hub.Publish(RealtimeEvent{Type: "driver.location", DriverID: driverB.ID, Lng: 116.5, Lat: 39.8, SpeedKPH: 28})
	select {
	case event := <-passengerSubscriber:
		t.Fatalf("did not expect passenger to receive other driver location, got %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
	select {
	case event := <-driverSubscriber:
		t.Fatalf("did not expect driver to receive other driver location, got %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestDriverRealtimeFilterAllowsCurrentDispatchOffer(t *testing.T) {
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

	order := app.Orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})
	attempt, err := app.Dispatch.Dispatch(order.ID)
	if err != nil {
		t.Fatalf("dispatch order: %v", err)
	}
	if attempt.DriverID != driverA.ID && attempt.DriverID != driverB.ID {
		t.Fatalf("expected offer to known driver, got %s", attempt.DriverID)
	}

	phone := "13900000001"
	if attempt.DriverID == driverB.ID {
		phone = "13900000002"
	}
	subscriber := app.Events.SubscribeFiltered(realtimeEventFilter(s, services.Session{Role: domain.RoleDriver, Phone: phone}))
	defer app.Events.Unsubscribe(subscriber)

	app.Events.Publish(RealtimeEvent{Type: "order.updated", OrderID: order.ID, Status: string(domain.OrderDispatching)})
	select {
	case event := <-subscriber:
		if event.OrderID != order.ID {
			t.Fatalf("expected offered driver to receive dispatching order event, got %+v", event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected offered driver to receive dispatching order event")
	}
}

func TestPassengerRealtimeFilterStopsDriverLocationAfterOrderCompletion(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	hub := NewEventHub()

	completedOrder := s.CreateOrder(domain.RideOrder{PassengerID: passenger.ID, DriverID: driver.ID, Status: domain.OrderCompleted, Pickup: domain.Point{Name: "A"}, Dropoff: domain.Point{Name: "B"}})
	subscriber := hub.SubscribeFiltered(realtimeEventFilter(s, services.Session{Role: domain.RolePassenger, Phone: "13800000001"}))
	defer hub.Unsubscribe(subscriber)

	hub.Publish(RealtimeEvent{Type: "order.updated", OrderID: completedOrder.ID, Status: string(domain.OrderCompleted)})
	if event := <-subscriber; event.OrderID != completedOrder.ID {
		t.Fatalf("expected passenger to still receive own completed order event, got %+v", event)
	}

	hub.Publish(RealtimeEvent{Type: "driver.location", DriverID: driver.ID, Lng: 116.4, Lat: 39.9, SpeedKPH: 20})
	select {
	case event := <-subscriber:
		t.Fatalf("did not expect passenger to receive driver location after order completion, got %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
}
