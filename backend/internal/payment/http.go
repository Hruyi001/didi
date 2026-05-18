package payment

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/payments/:orderID/pay", func(c *gin.Context) {
		payment, err := svc.Pay(c.Param("orderID"))
		if err != nil {
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"data": payment})
	})
}
