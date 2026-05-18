package http

import "github.com/gin-gonic/gin"

func (a *App) adminOrders(c *gin.Context) { ok(c, a.Store.ListOrders()) }
func (a *App) adminDrivers(c *gin.Context) { ok(c, a.Store.ListDrivers()) }
func (a *App) adminApproveDriver(c *gin.Context) {
	ok(c, gin.H{"driverId": c.Param("id"), "auditState": "APPROVED"})
}
