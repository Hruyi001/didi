package dispatch

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/dispatch/start", func(c *gin.Context) {
		attempt, err := svc.Start("order-1", []string{"driver-1"})
		if err != nil {
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"data": attempt})
	})
}
