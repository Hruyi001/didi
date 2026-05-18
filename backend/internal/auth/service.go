package auth

import (
	"context"
	"fmt"
	"time"

	"didi/backend/internal/contracts"
	"github.com/google/uuid"
)

type LoginRequest struct {
	Phone string                `json:"phone"`
	Code  string                `json:"code"`
	Role  contracts.AccountRole `json:"role"`
}

type LoginResult struct {
	AccessToken  string                `json:"accessToken"`
	RefreshToken string                `json:"refreshToken"`
	Role         contracts.AccountRole `json:"role"`
	ExpiresAt    time.Time             `json:"expiresAt"`
}

type Service struct{ repo *MemoryRepo }

func NewService(repo *MemoryRepo) *Service { return &Service{repo: repo} }

func (s *Service) Login(_ context.Context, req LoginRequest) (LoginResult, error) {
	if req.Code != "123456" {
		return LoginResult{}, fmt.Errorf("验证码错误")
	}
	return LoginResult{
		AccessToken:  "access-" + uuid.NewString(),
		RefreshToken: "refresh-" + uuid.NewString(),
		Role:         req.Role,
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}, nil
}
