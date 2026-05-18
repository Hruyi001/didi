package http

import (
	"didi/backend/internal/domain"
	"didi/backend/internal/services"
	"didi/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type App struct {
	Store    store.Store
	Auth     *services.AuthService
	Risk     *services.RiskService
	Orders   *services.OrderService
	Dispatch *services.DispatchService
	Payment  *services.PaymentService
	Events   *EventHub
}

func NewRouter(app *App) *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) { ok(c, gin.H{"status": "ok"}) })
	r.POST("/api/auth/send-code", app.sendCode)
	r.POST("/api/auth/login", app.login)

	passenger := r.Group("/api/passenger", app.requireAuth(domain.RolePassenger))
	passenger.POST("/orders", app.createPassengerOrder)
	passenger.GET("/orders", app.listOrders)
	passenger.POST("/orders/:id/cancel", app.cancelOrder)
	passenger.POST("/orders/:id/pay", app.payOrder)
	passenger.POST("/orders/:id/review", app.reviewOrder)
	passenger.GET("/orders/:id/driver-location", app.getPassengerDriverLocation)

	driver := r.Group("/api/driver", app.requireAuth(domain.RoleDriver))
	driver.GET("/profile", app.driverProfile)
	driver.GET("/orders", app.driverOrders)
	driver.POST("/online", app.driverOnline)
	driver.POST("/offline", app.driverOffline)
	driver.POST("/orders/:id/accept", app.acceptOrder)
	driver.POST("/orders/:id/reject", app.rejectOrder)
	driver.POST("/orders/:id/arrive", app.arriveOrder)
	driver.POST("/orders/:id/start", app.startTrip)
	driver.POST("/orders/:id/end", app.endTrip)
	driver.POST("/location", app.updateDriverLocation)

	admin := r.Group("/api/admin", app.requireAuth(domain.RoleAdmin))
	admin.GET("/orders", app.adminOrders)
	admin.GET("/drivers", app.adminDrivers)
	admin.POST("/drivers/:id/approve", app.adminApproveDriver)

	r.GET("/ws", app.websocket)
	return r
}
