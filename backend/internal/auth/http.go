package auth

import (
	"net/http"

	"didi/backend/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/auth/login", func(c *gin.Context) {
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
	})
}
