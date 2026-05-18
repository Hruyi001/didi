package http

import (
	"net/http"

	"didi/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type updateDriverLocationRequest struct {
	DriverID string  `json:"driverId"`
	Lng      float64 `json:"lng"`
	Lat      float64 `json:"lat"`
	SpeedKph int     `json:"speedKph"`
}

func currentDriver(c *gin.Context, a *App) (domain.DriverProfile, bool) {
	session := currentSession(c)
	driver, exists := a.Store.GetDriverByPhone(session.Phone)
	return driver, exists
}

func (a *App) driverProfile(c *gin.Context) {
	driver, exists := currentDriver(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "暂无司机")
		return
	}
	ok(c, driver)
}

func (a *App) driverOrders(c *gin.Context) {
	driver, exists := currentDriver(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "暂无司机")
		return
	}
	ok(c, a.Store.ListOrdersForDriver(driver.ID))
}

func (a *App) driverOrder(c *gin.Context) (domain.DriverProfile, domain.RideOrder, bool) {
	current, exists := currentDriver(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "暂无司机")
		return domain.DriverProfile{}, domain.RideOrder{}, false
	}
	order, exists := a.Store.GetOrder(c.Param("id"))
	if !exists {
		fail(c, http.StatusNotFound, "订单不存在")
		return domain.DriverProfile{}, domain.RideOrder{}, false
	}
	if order.DriverID != current.ID {
		fail(c, http.StatusForbidden, "无权操作该订单")
		return domain.DriverProfile{}, domain.RideOrder{}, false
	}
	return current, order, true
}

func (a *App) driverOnline(c *gin.Context) {
	current, exists := currentDriver(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "暂无司机")
		return
	}
	driver, err := a.Store.SetDriverWorkStatus(current.ID, domain.DriverOnlineIdle)
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	ok(c, driver)
}

func (a *App) driverOffline(c *gin.Context) {
	current, exists := currentDriver(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "暂无司机")
		return
	}
	driver, err := a.Store.SetDriverWorkStatus(current.ID, domain.DriverOffline)
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	ok(c, driver)
}

func (a *App) acceptOrder(c *gin.Context) {
	current, exists := currentDriver(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "暂无司机")
		return
	}
	attempts := a.Store.ListDispatchAttempts(c.Param("id"))
	if len(attempts) == 0 || attempts[len(attempts)-1].Status != domain.DispatchOffered || attempts[len(attempts)-1].DriverID != current.ID {
		fail(c, http.StatusForbidden, "当前订单未派给该司机")
		return
	}
	order, err := a.Dispatch.Accept(c.Param("id"), current.ID)
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	if a.Events != nil {
		a.Events.Publish(RealtimeEvent{Type: "order.updated", OrderID: order.ID, Status: string(order.Status), Message: "司机已接单"})
	}
	ok(c, order)
}

func (a *App) rejectOrder(c *gin.Context) {
	current, exists := currentDriver(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "暂无司机")
		return
	}
	attempts := a.Store.ListDispatchAttempts(c.Param("id"))
	if len(attempts) == 0 || attempts[len(attempts)-1].Status != domain.DispatchOffered || attempts[len(attempts)-1].DriverID != current.ID {
		fail(c, http.StatusForbidden, "当前订单未派给该司机")
		return
	}
	attempt, err := a.Dispatch.Reject(c.Param("id"), current.ID, "司机拒单")
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	if a.Events != nil && attempt.Status == domain.DispatchOffered {
		a.Events.Publish(RealtimeEvent{Type: "order.updated", OrderID: c.Param("id"), Status: string(domain.OrderDispatching), Message: "订单已重新派单"})
	}
	ok(c, attempt)
}

func (a *App) arriveOrder(c *gin.Context) {
	if _, _, ok := a.driverOrder(c); !ok {
		return
	}
	order, err := a.Orders.Arrive(c.Param("id"))
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	if a.Events != nil {
		a.Events.Publish(RealtimeEvent{Type: "order.updated", OrderID: order.ID, Status: string(order.Status), Message: "司机已到达上车点"})
	}
	ok(c, order)
}

func (a *App) startTrip(c *gin.Context) {
	if _, _, ok := a.driverOrder(c); !ok {
		return
	}
	order, err := a.Orders.StartTrip(c.Param("id"))
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	if a.Events != nil {
		a.Events.Publish(RealtimeEvent{Type: "order.updated", OrderID: order.ID, Status: string(order.Status), Message: "行程已开始"})
	}
	ok(c, order)
}

func (a *App) endTrip(c *gin.Context) {
	if _, _, ok := a.driverOrder(c); !ok {
		return
	}
	order, err := a.Orders.EndTrip(c.Param("id"))
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	if a.Events != nil {
		a.Events.Publish(RealtimeEvent{Type: "order.updated", OrderID: order.ID, Status: string(order.Status), Message: "行程已结束，等待支付"})
	}
	ok(c, order)
}

func (a *App) updateDriverLocation(c *gin.Context) {
	current, exists := currentDriver(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "暂无司机")
		return
	}
	var req updateDriverLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "司机和位置不能为空")
		return
	}
	location, err := a.Store.UpdateDriverLocation(current.ID, domain.DriverLocation{DriverID: current.ID, Lng: req.Lng, Lat: req.Lat, SpeedKPH: req.SpeedKph})
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	if a.Events != nil {
		a.Events.Publish(RealtimeEvent{Type: "driver.location", DriverID: location.DriverID, Lng: location.Lng, Lat: location.Lat, SpeedKPH: location.SpeedKPH, Message: "司机位置已更新"})
	}
	ok(c, location)
}
