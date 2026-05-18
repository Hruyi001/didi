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

func TestLoginAcceptsAllSeededRoles(t *testing.T) {
	svc := NewService(NewMemoryRepo())
	cases := []LoginRequest{
		{Phone: "13800000001", Code: "123456", Role: contracts.RolePassenger},
		{Phone: "13900000001", Code: "123456", Role: contracts.RoleDriver},
		{Phone: "13700000001", Code: "123456", Role: contracts.RoleAdmin},
	}
	for _, req := range cases {
		if _, err := svc.Login(context.Background(), req); err != nil {
			t.Fatalf("expected login success for %s: %v", req.Role, err)
		}
	}
}

func TestLoginRejectsPhoneRoleMismatch(t *testing.T) {
	svc := NewService(NewMemoryRepo())
	cases := []LoginRequest{
		{Phone: "13800000001", Code: "123456", Role: contracts.RoleDriver},
		{Phone: "13900000001", Code: "123456", Role: contracts.RolePassenger},
		{Phone: "13700000002", Code: "123456", Role: contracts.RoleAdmin},
	}
	for _, req := range cases {
		if _, err := svc.Login(context.Background(), req); err == nil || err.Error() != "手机号与登录角色不匹配" {
			t.Fatalf("expected mismatch error for %+v, got %v", req, err)
		}
	}
}
