package http

import (
	"net/http"

	"didi/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type createOrderRequest struct {
	PassengerID string       `json:"passengerId"`
	Pickup      domain.Point `json:"pickup"`
	Dropoff     domain.Point `json:"dropoff"`
}

type reviewRequest struct {
	Score   int    `json:"score"`
	Content string `json:"content"`
}

func currentPassenger(c *gin.Context, a *App) (domain.PassengerProfile, bool) {
	session := currentSession(c)
	passenger, exists := a.Store.GetPassengerByPhone(session.Phone)
	return passenger, exists
}

func (a *App) passengerOrder(c *gin.Context) (domain.RideOrder, bool) {
	passenger, exists := currentPassenger(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "乘客不存在")
		return domain.RideOrder{}, false
	}
	order, exists := a.Store.GetOrder(c.Param("id"))
	if !exists {
		fail(c, http.StatusNotFound, "订单不存在")
		return domain.RideOrder{}, false
	}
	if order.PassengerID != passenger.ID {
		fail(c, http.StatusForbidden, "无权访问该订单")
		return domain.RideOrder{}, false
	}
	return order, true
}

func (a *App) createPassengerOrder(c *gin.Context) {
	passenger, exists := currentPassenger(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "乘客不存在")
		return
	}
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "乘客和起终点不能为空")
		return
	}
	order := a.Orders.CreateRide(passenger.ID, req.Pickup, req.Dropoff)
	attempt, err := a.Dispatch.Dispatch(order.ID)
	if err != nil {
		if a.Events != nil {
			a.Events.Publish(RealtimeEvent{Type: "order.updated", OrderID: order.ID, Status: string(order.Status), Message: err.Error()})
		}
		ok(c, gin.H{"order": order, "dispatchError": err.Error()})
		return
	}
	if a.Events != nil {
		a.Events.Publish(RealtimeEvent{Type: "order.updated", OrderID: order.ID, Status: string(domain.OrderDispatching), Message: "订单已创建并开始派单"})
	}
	ok(c, gin.H{"order": order, "dispatchAttempt": attempt})
}

func (a *App) listOrders(c *gin.Context) {
	passenger, exists := currentPassenger(c, a)
	if !exists {
		fail(c, http.StatusNotFound, "乘客不存在")
		return
	}
	ok(c, a.Store.ListOrdersByPassenger(passenger.ID))
}

func (a *App) cancelOrder(c *gin.Context) {
	if _, ok := a.passengerOrder(c); !ok {
		return
	}
	order, err := a.Orders.Cancel(c.Param("id"), "乘客取消")
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	ok(c, order)
}

func (a *App) payOrder(c *gin.Context) {
	if _, ok := a.passengerOrder(c); !ok {
		return
	}
	payment, order, err := a.Payment.Pay(c.Param("id"))
	if err != nil {
		fail(c, http.StatusConflict, err.Error())
		return
	}
	if a.Events != nil {
		a.Events.Publish(RealtimeEvent{Type: "order.updated", OrderID: order.ID, Status: string(order.Status), Message: "订单已支付完成"})
	}
	ok(c, gin.H{"payment": payment, "order": order})
}

func (a *App) reviewOrder(c *gin.Context) {
	if _, ok := a.passengerOrder(c); !ok {
		return
	}
	var req reviewRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Score < 1 || req.Score > 5 {
		fail(c, http.StatusBadRequest, "评分必须为 1-5")
		return
	}
	ok(c, a.Orders.Review(c.Param("id"), req.Score, req.Content))
}

func (a *App) getPassengerDriverLocation(c *gin.Context) {
	order, allowed := a.passengerOrder(c)
	if !allowed {
		return
	}
	if order.Status != domain.OrderWaitingPickup && order.Status != domain.OrderDriverArrived && order.Status != domain.OrderInProgress {
		fail(c, http.StatusConflict, "当前订单状态不可查看司机位置")
		return
	}
	if order.DriverID == "" {
		fail(c, http.StatusConflict, "订单尚未分配司机")
		return
	}
	location, exists := a.Store.GetDriverLocation(order.DriverID)
	if !exists {
		fail(c, http.StatusNotFound, "暂无司机位置")
		return
	}
	ok(c, location)
}
