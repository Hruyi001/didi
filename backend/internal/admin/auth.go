package admin

import (
	"net/http"
	"strings"

	"didi/backend/internal/domain"
	"didi/backend/internal/platform/httpx"
	"didi/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func requireAdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") {
			httpx.Fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		session, err := services.ValidateToken(strings.TrimPrefix(authorization, "Bearer "))
		if err != nil {
			httpx.Fail(c, http.StatusUnauthorized, "token error")
			c.Abort()
			return
		}
		if session.Role != domain.RoleAdmin {
			httpx.Fail(c, http.StatusForbidden, "无权访问该资源")
			c.Abort()
			return
		}
		c.Next()
	}
}
