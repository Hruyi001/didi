package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestPassengerSendCodeViaGateway(t *testing.T) {
	payload := []byte(`{"phone":"13800000001"}`)
	res, err := http.Post("http://localhost:8080/api/auth/send-code", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("expected send-code success: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data field in send-code response")
	}
	if data["message"] == "" {
		t.Fatal("expected send-code message")
	}
}

func TestPassengerSendCodeRateLimitViaGateway(t *testing.T) {
	payload := []byte(`{"phone":"13800000009"}`)
	for i := 0; i < 5; i++ {
		res, err := http.Post("http://localhost:8080/api/auth/send-code", "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("request %d expected success: %v", i+1, err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("request %d expected 200, got %d", i+1, res.StatusCode)
		}
	}
	res, err := http.Post("http://localhost:8080/api/auth/send-code", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("expected rate-limit response: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", res.StatusCode)
	}
}

func TestPassengerLoginViaGateway(t *testing.T) {
	payload := []byte(`{"phone":"13800000001","code":"123456","role":"PASSENGER"}`)
	res, err := http.Post("http://localhost:8080/api/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("expected login success: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if _, ok := body["data"]; !ok {
		t.Fatal("expected data field in gateway response")
	}
}

func TestPassengerTokenWorksAgainstLegacyProtectedEndpoint(t *testing.T) {
	token := loginViaGateway(t, "13800000001", "PASSENGER")
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/api/passenger/orders", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	protectedRes, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected protected endpoint success: %v", err)
	}
	defer protectedRes.Body.Close()
	if protectedRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from protected endpoint, got %d", protectedRes.StatusCode)
	}
}

func TestAdminDriversReadSharedStateViaGateway(t *testing.T) {
	token := loginViaGateway(t, "13700000001", "ADMIN")
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/api/admin/drivers", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected admin drivers success: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Data []struct {
			ID         string `json:"id"`
			Phone      string `json:"phone"`
			AuditState string `json:"auditState"`
			WorkStatus string `json:"workStatus"`
			PlateNo    string `json:"plateNo"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if len(body.Data) == 0 {
		t.Fatalf("expected seeded shared drivers, got %#v", body.Data)
	}
	found := false
	for _, driver := range body.Data {
		if driver.Phone == "13900000001" && driver.AuditState == "APPROVED" && driver.PlateNo == "京A12345" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected admin-service to read api-seeded driver from MySQL, got %#v", body.Data)
	}
}

func TestPassengerCannotReadAdminDriversViaGateway(t *testing.T) {
	token := loginViaGateway(t, "13800000001", "PASSENGER")
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/api/admin/drivers", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected forbidden response: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.StatusCode)
	}
}

func loginViaGateway(t *testing.T, phone, role string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"phone": phone, "code": "123456", "role": role})
	if err != nil {
		t.Fatalf("marshal login payload: %v", err)
	}
	res, err := http.Post("http://localhost:8080/api/auth/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("expected login success: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body.Data.AccessToken == "" {
		t.Fatal("expected access token")
	}
	return body.Data.AccessToken
}
