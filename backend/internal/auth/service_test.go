package auth

import (
	"context"
	"testing"

	"didi/backend/internal/contracts"
)

func TestLoginAcceptsFixedCode(t *testing.T) {
	svc := NewService(NewMemoryRepo())
	result, err := svc.Login(context.Background(), LoginRequest{Phone: "13800000001", Code: "123456", Role: contracts.RolePassenger})
	if err != nil {
		t.Fatalf("expected login success: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatal("expected both tokens")
	}
}
