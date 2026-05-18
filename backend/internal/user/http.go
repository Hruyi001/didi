package user

import "github.com/gin-gonic/gin"

func RegisterHTTP(r *gin.Engine, svc *Service) {
	r.POST("/v1/users/passengers/ensure", func(c *gin.Context) {
		profile := svc.EnsurePassenger("acct-demo", "13800000001")
		c.JSON(200, gin.H{"data": profile})
	})
}
