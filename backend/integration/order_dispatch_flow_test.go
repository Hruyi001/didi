package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

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
	payload := []byte(`{"phone":"13800000001","code":"123456","role":"PASSENGER"}`)
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
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/api/passenger/orders", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+body.Data.AccessToken)
	protectedRes, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("expected protected endpoint success: %v", err)
	}
	defer protectedRes.Body.Close()
	if protectedRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from protected endpoint, got %d", protectedRes.StatusCode)
	}
}
