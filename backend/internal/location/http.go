package location

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.GET("/v1/location/nearby", func(c *gin.Context) {
		results := svc.Nearby(116.398, 39.900, 10)
		c.JSON(200, gin.H{"data": results})
	})
}
