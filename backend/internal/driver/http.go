package driver

import (
	"net/http"

	"didi/backend/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/drivers/:id/online", func(c *gin.Context) {
		driver, err := svc.SetOnline(c.Param("id"))
		if err != nil {
			httpx.Fail(c, http.StatusConflict, err.Error())
			return
		}
		httpx.OK(c, driver)
	})
}
