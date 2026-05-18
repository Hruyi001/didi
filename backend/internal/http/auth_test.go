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

func TestPassengerOrdersRequireAccessToken(t *testing.T) {
	s := store.NewMemoryStore()
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	req := httptest.NewRequest(http.MethodGet, "/api/passenger/orders", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestPassengerOrdersRejectDriverRole(t *testing.T) {
	s := store.NewMemoryStore()
	s.SeedApprovedDriver("13900000001", "京A12345")
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)
	token := loginTokenForPhone(t, router, domain.RoleDriver, "13900000001")

	req := httptest.NewRequest(http.MethodGet, "/api/passenger/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestPassengerOrdersAllowPassengerRole(t *testing.T) {
	s := store.NewMemoryStore()
	s.SeedPassenger("13800000001")
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)
	token := loginToken(t, router, domain.RolePassenger)

	req := httptest.NewRequest(http.MethodGet, "/api/passenger/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestDriverProfileRejectsPassengerRole(t *testing.T) {
	s := store.NewMemoryStore()
	s.SeedPassenger("13800000001")
	s.SeedApprovedDriver("13900000001", "京A12345")
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)
	token := loginTokenForPhone(t, router, domain.RolePassenger, "13800000001")

	req := httptest.NewRequest(http.MethodGet, "/api/driver/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", res.Code, res.Body.String())
	}
}

func TestLoginRejectsPhoneRoleMismatch(t *testing.T) {
	s := store.NewMemoryStore()
	s.SeedPassenger("13800000001")
	s.SeedApprovedDriver("13900000001", "京A12345")
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	tests := []struct {
		name string
		body map[string]any
	}{
		{name: "passenger-phone-as-driver", body: map[string]any{"phone": "13800000001", "code": "123456", "role": domain.RoleDriver}},
		{name: "driver-phone-as-passenger", body: map[string]any{"phone": "13900000001", "code": "123456", "role": domain.RolePassenger}},
		{name: "unknown-admin-phone", body: map[string]any{"phone": "13700000002", "code": "123456", "role": domain.RoleAdmin}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d body=%s", res.Code, res.Body.String())
			}
		})
	}
}

func TestAdminLoginAllowsDemoAdminPhone(t *testing.T) {
	s := store.NewMemoryStore()
	app := &App{Store: s, Auth: services.NewAuthService(), Risk: services.NewRiskService(), Orders: services.NewOrderService(s), Dispatch: services.NewDispatchService(s), Payment: services.NewPaymentService(s), Events: NewEventHub()}
	router := NewRouter(app)

	payload, _ := json.Marshal(map[string]any{"phone": "13700000001", "code": "123456", "role": domain.RoleAdmin})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.Code, res.Body.String())
	}
}

func loginToken(t *testing.T, router http.Handler, role domain.AccountRole) string {
	t.Helper()
	body := map[string]any{"phone": "13800000001", "code": "123456", "role": role}
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
