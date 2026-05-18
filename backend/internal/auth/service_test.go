package auth

import (
	"context"
	"testing"

	"didi/backend/internal/contracts"
	"didi/backend/internal/services"
)

func TestSendCodeReturnsMessage(t *testing.T) {
	svc := NewService(NewMemoryRepo(), services.NewRiskService(), services.NewAuthService())
	message, err := svc.SendCode("13800000001")
	if err != nil {
		t.Fatalf("expected send-code success: %v", err)
	}
	if message == "" {
		t.Fatal("expected send-code message")
	}
}

func TestSendCodeRejectsEmptyPhone(t *testing.T) {
	svc := NewService(NewMemoryRepo(), services.NewRiskService(), services.NewAuthService())
	if _, err := svc.SendCode(""); err == nil || err.Error() != "手机号不能为空" {
		t.Fatalf("expected empty phone error, got %v", err)
	}
}

func TestSendCodeRateLimitsAfterFiveRequests(t *testing.T) {
	svc := NewService(NewMemoryRepo(), services.NewRiskService(), services.NewAuthService())
	for i := 0; i < 5; i++ {
		if _, err := svc.SendCode("13800000001"); err != nil {
			t.Fatalf("request %d expected success: %v", i+1, err)
		}
	}
	if _, err := svc.SendCode("13800000001"); err == nil || err.Error() != "短信发送过于频繁" {
		t.Fatalf("expected rate-limit error, got %v", err)
	}
}

func TestLoginAcceptsFixedCode(t *testing.T) {
	svc := NewService(NewMemoryRepo(), services.NewRiskService(), services.NewAuthService())
	result, err := svc.Login(context.Background(), LoginRequest{Phone: "13800000001", Code: "123456", Role: contracts.RolePassenger})
	if err != nil {
		t.Fatalf("expected login success: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatal("expected both tokens")
	}
}

func TestLoginAcceptsAllSeededRoles(t *testing.T) {
	svc := NewService(NewMemoryRepo(), services.NewRiskService(), services.NewAuthService())
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
	svc := NewService(NewMemoryRepo(), services.NewRiskService(), services.NewAuthService())
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
