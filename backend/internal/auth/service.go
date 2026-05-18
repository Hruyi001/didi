package auth

import (
	"context"
	"fmt"
	"time"

	"didi/backend/internal/contracts"
	"didi/backend/internal/domain"
	"didi/backend/internal/services"
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

type Service struct {
	repo *MemoryRepo
	risk *services.RiskService
	sms  *services.AuthService
}

func NewService(repo *MemoryRepo, risk *services.RiskService, sms *services.AuthService) *Service {
	return &Service{repo: repo, risk: risk, sms: sms}
}

func (s *Service) SendCode(phone string) (string, error) {
	if phone == "" {
		return "", fmt.Errorf("手机号不能为空")
	}
	if err := s.risk.CheckSMS(phone); err != nil {
		return "", err
	}
	return s.sms.SendCode(phone), nil
}

func (s *Service) Login(_ context.Context, req LoginRequest) (LoginResult, error) {
	if !s.repo.IdentityExists(req.Phone, req.Role) {
		return LoginResult{}, fmt.Errorf("手机号与登录角色不匹配")
	}
	accessToken, refreshToken, expiresAt, err := services.IssueTokens(req.Phone, domain.AccountRole(req.Role))
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Role:         req.Role,
		ExpiresAt:    expiresAt,
	}, nil
}
