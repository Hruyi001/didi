package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

type tokenClaims struct {
	Phone     string             `json:"phone"`
	Role      domain.AccountRole `json:"role"`
	ExpiresAt time.Time          `json:"expiresAt"`
}

func NewAuthService() *AuthService { return &AuthService{sessions: map[string]Session{}} }

func (s *AuthService) SendCode(phone string) string {
	return fmt.Sprintf("验证码已发送到 %s，演示验证码固定为 123456", phone)
}

func (s *AuthService) Login(phone, code string, role domain.AccountRole) (LoginResult, error) {
	if code != "123456" {
		return LoginResult{}, fmt.Errorf("验证码错误")
	}
	accessToken, refreshToken, expiresAt, err := IssueTokens(phone, role)
	if err != nil {
		return LoginResult{}, err
	}
	s.sessions[accessToken] = Session{AccessToken: accessToken, Role: role, Phone: phone, ExpiresAt: expiresAt}
	return LoginResult{AccessToken: accessToken, RefreshToken: refreshToken, Role: role, ExpiresAt: expiresAt}, nil
}

func (s *AuthService) ValidateAccessToken(token string) (Session, error) {
	session, err := ValidateToken(token)
	if err != nil {
		return Session{}, err
	}
	if s.sessions != nil {
		s.sessions[token] = session
	}
	return session, nil
}

func IssueTokens(phone string, role domain.AccountRole) (string, string, time.Time, error) {
	expiresAt := time.Now().Add(2 * time.Hour)
	accessToken, err := signToken(tokenClaims{Phone: phone, Role: role, ExpiresAt: expiresAt})
	if err != nil {
		return "", "", time.Time{}, err
	}
	refreshToken := "refresh-" + uuid.NewString()
	return accessToken, refreshToken, expiresAt, nil
}

func ValidateToken(token string) (Session, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Session{}, fmt.Errorf("登录态无效")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Session{}, fmt.Errorf("登录态无效")
	}
	expectedSig := signPayload(payload)
	if !hmac.Equal([]byte(parts[1]), []byte(expectedSig)) {
		return Session{}, fmt.Errorf("登录态无效")
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Session{}, fmt.Errorf("登录态无效")
	}
	if time.Now().After(claims.ExpiresAt) {
		return Session{}, fmt.Errorf("登录态已过期")
	}
	return Session{AccessToken: token, Role: claims.Role, Phone: claims.Phone, ExpiresAt: claims.ExpiresAt}, nil
}

func signToken(claims tokenClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + signPayload(payload), nil
}

func signPayload(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(tokenSecret()))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func tokenSecret() string {
	if value := os.Getenv("AUTH_TOKEN_SECRET"); value != "" {
		return value
	}
	return "didi-demo-shared-secret"
}
