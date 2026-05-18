package auth

import (
	"net/http"

	"didi/backend/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.GET("/health", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok", "service": "auth-service"})
	})

	sendCodeHandler := func(c *gin.Context) {
		var req struct {
			Phone string `json:"phone"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, http.StatusBadRequest, "手机号不能为空")
			return
		}
		message, err := svc.SendCode(req.Phone)
		if err != nil {
			status := http.StatusTooManyRequests
			if err.Error() == "手机号不能为空" {
				status = http.StatusBadRequest
			}
			httpx.Fail(c, status, err.Error())
			return
		}
		httpx.OK(c, gin.H{"message": message})
	}

	loginHandler := func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.Fail(c, http.StatusBadRequest, "请求格式错误")
			return
		}
		result, err := svc.Login(c.Request.Context(), req)
		if err != nil {
			httpx.Fail(c, http.StatusUnauthorized, err.Error())
			return
		}
		httpx.OK(c, result)
	}

	r.POST("/v1/auth/send-code", sendCodeHandler)
	r.POST("/api/auth/send-code", sendCodeHandler)
	r.POST("/v1/auth/login", loginHandler)
	r.POST("/api/auth/login", loginHandler)
}
