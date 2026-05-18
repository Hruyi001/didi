package services

import (
	"fmt"
	"time"

	"didi/backend/internal/domain"
	"github.com/google/uuid"
)

type LoginResult struct {
	AccessToken  string             `json:"accessToken"`
	RefreshToken string             `json:"refreshToken"`
	Role         domain.AccountRole `json:"role"`
	ExpiresAt    time.Time          `json:"expiresAt"`
}

type Session struct {
	AccessToken string
	Role        domain.AccountRole
	Phone       string
	ExpiresAt   time.Time
}

type AuthService struct {
	sessions map[string]Session
}

func NewAuthService() *AuthService { return &AuthService{sessions: map[string]Session{}} }

func (s *AuthService) SendCode(phone string) string {
	return fmt.Sprintf("验证码已发送到 %s，演示验证码固定为 123456", phone)
}

func (s *AuthService) Login(phone, code string, role domain.AccountRole) (LoginResult, error) {
	if code != "123456" {
		return LoginResult{}, fmt.Errorf("验证码错误")
	}
	accessToken := "access-" + uuid.NewString()
	refreshToken := "refresh-" + uuid.NewString()
	expiresAt := time.Now().Add(2 * time.Hour)
	s.sessions[accessToken] = Session{AccessToken: accessToken, Role: role, Phone: phone, ExpiresAt: expiresAt}
	return LoginResult{AccessToken: accessToken, RefreshToken: refreshToken, Role: role, ExpiresAt: expiresAt}, nil
}

func (s *AuthService) ValidateAccessToken(token string) (Session, error) {
	session, ok := s.sessions[token]
	if !ok {
		return Session{}, fmt.Errorf("登录态无效")
	}
	if time.Now().After(session.ExpiresAt) {
		delete(s.sessions, token)
		return Session{}, fmt.Errorf("登录态已过期")
	}
	return session, nil
}
