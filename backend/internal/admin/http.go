package admin

import (
	"log"
	"net/http"

	"didi/backend/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.GET("/health", func(c *gin.Context) {
		httpx.OK(c, gin.H{"status": "ok", "service": "admin-service"})
	})

	handler := func(c *gin.Context) {
		drivers, err := svc.ListDrivers()
		if err != nil {
			log.Printf("admin list drivers failed: %v", err)
			httpx.Fail(c, http.StatusInternalServerError, "查询司机列表失败")
			return
		}
		httpx.OK(c, drivers)
	}

	admin := r.Group("/api/admin", requireAdminAuth())
	admin.GET("/drivers", handler)

	v1 := r.Group("/v1/admin", requireAdminAuth())
	v1.GET("/drivers", handler)
}
