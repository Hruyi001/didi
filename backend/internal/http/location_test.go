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

func TestDriverLocationUpdatePublishesRealtimeEvent(t *testing.T) {
	s := store.NewMemoryStore()
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	hub := NewEventHub()
	subscriber := hub.Subscribe()
	defer hub.Unsubscribe(subscriber)
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: hub}
	router := NewRouter(app)

	body := map[string]any{"driverId": driver.ID, "lng": 116.397, "lat": 39.908, "speedKph": 35}
	payload, _ := json.Marshal(body)
	driverToken := loginTokenForPhone(t, router, domain.RoleDriver, "13900000001")
	req := httptest.NewRequest(http.MethodPost, "/api/driver/location", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+driverToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}

	select {
	case event := <-subscriber:
		if event.Type != "driver.location" {
			t.Fatalf("expected driver.location, got %s", event.Type)
		}
		if event.DriverID != driver.ID {
			t.Fatalf("expected driver id %s, got %s", driver.ID, event.DriverID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected realtime driver location event")
	}
}

func TestPassengerCanReadAssignedDriverLocation(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	if _, err := s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver online: %v", err)
	}
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	order := app.Orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})
	if _, err := app.Dispatch.Dispatch(order.ID); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if _, err := app.Dispatch.Accept(order.ID, driver.ID); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if _, err := s.UpdateDriverLocation(driver.ID, domain.DriverLocation{DriverID: driver.ID, Lng: 116.4, Lat: 39.9, SpeedKPH: 40}); err != nil {
		t.Fatalf("update driver location: %v", err)
	}

	passengerToken := loginTokenForPhone(t, router, domain.RolePassenger, "13800000001")
	req := httptest.NewRequest(http.MethodGet, "/api/passenger/orders/"+order.ID+"/driver-location", nil)
	req.Header.Set("Authorization", "Bearer "+passengerToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestPassengerCannotReadDriverLocationForCompletedOrder(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	if _, err := s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver online: %v", err)
	}
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	order := app.Orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})
	if _, err := app.Dispatch.Dispatch(order.ID); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if _, err := app.Dispatch.Accept(order.ID, driver.ID); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if _, err := app.Orders.Arrive(order.ID); err != nil {
		t.Fatalf("arrive: %v", err)
	}
	if _, err := app.Orders.StartTrip(order.ID); err != nil {
		t.Fatalf("start trip: %v", err)
	}
	if _, err := app.Orders.EndTrip(order.ID); err != nil {
		t.Fatalf("end trip: %v", err)
	}
	if _, _, err := app.Payment.Pay(order.ID); err != nil {
		t.Fatalf("pay: %v", err)
	}
	if _, err := s.UpdateDriverLocation(driver.ID, domain.DriverLocation{DriverID: driver.ID, Lng: 116.4, Lat: 39.9, SpeedKPH: 40}); err != nil {
		t.Fatalf("update driver location: %v", err)
	}

	passengerToken := loginTokenForPhone(t, router, domain.RolePassenger, "13800000001")
	req := httptest.NewRequest(http.MethodGet, "/api/passenger/orders/"+order.ID+"/driver-location", nil)
	req.Header.Set("Authorization", "Bearer "+passengerToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusConflict {
		t.Fatalf("expected 409 for completed order, got %d body=%s", res.Code, res.Body.String())
	}
}
