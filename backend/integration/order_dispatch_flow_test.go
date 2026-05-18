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
