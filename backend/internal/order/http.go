package order

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/orders", func(c *gin.Context) {
		order := svc.Create("passenger-1", "pickup", "dropoff")
		c.JSON(200, gin.H{"data": order})
	})
}
