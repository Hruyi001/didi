package http

import (
	"net/http"
	"time"

	"didi/backend/internal/domain"
	"didi/backend/internal/services"
	"didi/backend/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func realtimeEventFilter(s store.Store, session services.Session) func(RealtimeEvent) bool {
	switch session.Role {
	case domain.RoleAdmin:
		return func(RealtimeEvent) bool { return true }
	case domain.RolePassenger:
		passenger, exists := s.GetPassengerByPhone(session.Phone)
		if !exists {
			return func(RealtimeEvent) bool { return false }
		}
		return func(event RealtimeEvent) bool {
			switch event.Type {
			case "order.updated":
				order, exists := s.GetOrder(event.OrderID)
				return exists && order.PassengerID == passenger.ID
			case "driver.location":
				for _, order := range s.ListOrdersByPassenger(passenger.ID) {
					if order.DriverID != event.DriverID {
						continue
					}
					if order.Status == domain.OrderWaitingPickup || order.Status == domain.OrderDriverArrived || order.Status == domain.OrderInProgress {
						return true
					}
				}
				return false
			default:
				return false
			}
		}
	case domain.RoleDriver:
		driver, exists := s.GetDriverByPhone(session.Phone)
		if !exists {
			return func(RealtimeEvent) bool { return false }
		}
		return func(event RealtimeEvent) bool {
			switch event.Type {
			case "order.updated":
				order, exists := s.GetOrder(event.OrderID)
				if !exists {
					return false
				}
				if order.DriverID == driver.ID {
					return true
				}
				attempts := s.ListDispatchAttempts(order.ID)
				if len(attempts) == 0 {
					return false
				}
				latest := attempts[len(attempts)-1]
				return latest.Status == domain.DispatchOffered && latest.DriverID == driver.ID
			case "driver.location":
				return event.DriverID == driver.ID
			default:
				return false
			}
		}
	default:
		return func(RealtimeEvent) bool { return false }
	}
}

func (a *App) websocket(c *gin.Context) {
	token := c.Query("token")
	session, err := a.Auth.ValidateAccessToken(token)
	if err != nil {
		fail(c, http.StatusUnauthorized, err.Error())
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	var subscriber chan RealtimeEvent
	if a.Events != nil {
		subscriber = a.Events.SubscribeFiltered(realtimeEventFilter(a.Store, session))
		defer a.Events.Unsubscribe(subscriber)
	}
	conn.WriteJSON(gin.H{"type": "CONNECTED", "message": "实时通道已连接"})
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := conn.WriteJSON(gin.H{"type": "HEARTBEAT", "time": time.Now()}); err != nil {
				return
			}
		case event, ok := <-subscriber:
			if !ok {
				return
			}
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		}
	}
}
