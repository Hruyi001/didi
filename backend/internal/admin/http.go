package admin

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.GET("/v1/admin/drivers", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": svc.ListDrivers()})
	})
}
