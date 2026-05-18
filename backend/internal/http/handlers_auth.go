package http

import (
	"net/http"

	"didi/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type sendCodeRequest struct{ Phone string `json:"phone"` }
type loginRequest struct {
	Phone string             `json:"phone"`
	Code  string             `json:"code"`
	Role  domain.AccountRole `json:"role"`
}

func (a *App) sendCode(c *gin.Context) {
	var req sendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Phone == "" {
		fail(c, http.StatusBadRequest, "手机号不能为空")
		return
	}
	if err := a.Risk.CheckSMS(req.Phone); err != nil {
		fail(c, http.StatusTooManyRequests, err.Error())
		return
	}
	ok(c, gin.H{"message": a.Auth.SendCode(req.Phone)})
}

func (a *App) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "请求格式错误")
		return
	}
	if !a.loginIdentityExists(req.Phone, req.Role) {
		fail(c, http.StatusUnauthorized, "手机号与登录角色不匹配")
		return
	}
	result, err := a.Auth.Login(req.Phone, req.Code, req.Role)
	if err != nil {
		fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	ok(c, result)
}

func (a *App) loginIdentityExists(phone string, role domain.AccountRole) bool {
	switch role {
	case domain.RolePassenger:
		_, exists := a.Store.GetPassengerByPhone(phone)
		return exists
	case domain.RoleDriver:
		_, exists := a.Store.GetDriverByPhone(phone)
		return exists
	case domain.RoleAdmin:
		return phone == "13700000001"
	default:
		return false
	}
}
