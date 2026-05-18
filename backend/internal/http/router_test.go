package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"didi/backend/internal/domain"
	"didi/backend/internal/services"
	"didi/backend/internal/store"
)

func TestPassengerCreateOrderAPI(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle)
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s)}
	router := NewRouter(app)

	body := map[string]any{
		"passengerId": passenger.ID,
		"pickup":      map[string]any{"name": "上车点", "lng": 116.397, "lat": 39.908},
		"dropoff":     map[string]any{"name": "下车点", "lng": 116.407, "lat": 39.918},
	}
	payload, _ := json.Marshal(body)
	passengerToken := loginToken(t, router, domain.RolePassenger)
	req := httptest.NewRequest(http.MethodPost, "/api/passenger/orders", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+passengerToken)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
	if len(s.ListOrders()) != 1 {
		t.Fatalf("expected one created order")
	}
}
