package http

import (
	"net/http"
	"strings"

	"didi/backend/internal/domain"
	"didi/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func ok(c *gin.Context, data any)                     { c.JSON(http.StatusOK, gin.H{"data": data}) }
func fail(c *gin.Context, status int, message string) { c.JSON(status, gin.H{"error": message}) }

func (a *App) requireAuth(roles ...domain.AccountRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if !strings.HasPrefix(authorization, "Bearer ") {
			fail(c, http.StatusUnauthorized, "请先登录")
			c.Abort()
			return
		}
		token := strings.TrimPrefix(authorization, "Bearer ")
		session, err := a.Auth.ValidateAccessToken(token)
		if err != nil {
			fail(c, http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		if len(roles) > 0 {
			allowed := false
			for _, role := range roles {
				if session.Role == role {
					allowed = true
					break
				}
			}
			if !allowed {
				fail(c, http.StatusForbidden, "无权访问该资源")
				c.Abort()
				return
			}
		}
		c.Set("session", session)
		c.Next()
	}
}

func currentSession(c *gin.Context) services.Session {
	value, _ := c.Get("session")
	session, _ := value.(services.Session)
	return session
}
