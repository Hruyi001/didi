package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	r := NewRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestLoginRoutesToAuthProxyWhenConfigured(t *testing.T) {
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "auth")
	}))
	defer auth.Close()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL, AuthBase: auth.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodPost, server.URL+"/api/auth/login")
	if body != "auth" {
		t.Fatalf("expected auth upstream, got %q", body)
	}
}

func TestLoginFallsBackToAPIProxyWithoutAuthBase(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodPost, server.URL+"/api/auth/login")
	if body != "api" {
		t.Fatalf("expected api upstream, got %q", body)
	}
}

func TestSendCodeRoutesToAuthProxyWhenConfigured(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "auth")
	}))
	defer auth.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL, AuthBase: auth.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodPost, server.URL+"/api/auth/send-code")
	if body != "auth" {
		t.Fatalf("expected auth upstream, got %q", body)
	}
}

func TestPassengerOrdersStillRouteToLegacyAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "auth")
	}))
	defer auth.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL, AuthBase: auth.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodGet, server.URL+"/api/passenger/orders")
	if body != "api" {
		t.Fatalf("expected api upstream, got %q", body)
	}
}

func TestAdminDriversRoutesToAdminProxyWhenConfigured(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "admin")
	}))
	defer admin.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL, AdminBase: admin.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodGet, server.URL+"/api/admin/drivers")
	if body != "admin" {
		t.Fatalf("expected admin upstream, got %q", body)
	}
}

func TestAdminDriversFallsBackToAPIProxyWithoutAdminBase(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodGet, server.URL+"/api/admin/drivers")
	if body != "api" {
		t.Fatalf("expected api upstream, got %q", body)
	}
}

func TestAdminApproveDriverStillRoutesToLegacyAPI(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "api")
	}))
	defer api.Close()
	admin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "admin")
	}))
	defer admin.Close()

	server := httptest.NewServer(NewRouter(&Clients{APIBase: api.URL, AdminBase: admin.URL}))
	defer server.Close()

	body := requestBody(t, http.MethodPost, server.URL+"/api/admin/drivers/driver-1/approve")
	if body != "api" {
		t.Fatalf("expected api upstream, got %q", body)
	}
}

func requestBody(t *testing.T, method, url string) string {
	t.Helper()

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", res.StatusCode, string(body))
	}
	return string(body)
}
