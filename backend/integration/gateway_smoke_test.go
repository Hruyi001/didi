package integration

import (
	"net/http"
	"testing"
)

func TestGatewayHealth(t *testing.T) {
	res, err := http.Get("http://localhost:8080/health")
	if err != nil {
		t.Fatalf("expected gateway to answer: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}
