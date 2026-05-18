package services

import (
	"fmt"

	"didi/backend/internal/domain"
	"didi/backend/internal/store"
)

type OrderService struct{ store store.Store }

func NewOrderService(store store.Store) *OrderService { return &OrderService{store: store} }

func (s *OrderService) CreateRide(passengerID string, pickup, dropoff domain.Point) domain.RideOrder {
	return s.store.CreateOrder(domain.RideOrder{PassengerID: passengerID, Pickup: pickup, Dropoff: dropoff})
}

func (s *OrderService) Arrive(orderID string) (domain.RideOrder, error) {
	return s.store.UpdateOrderStatus(orderID, domain.OrderDriverArrived)
}

func (s *OrderService) StartTrip(orderID string) (domain.RideOrder, error) {
	return s.store.UpdateOrderStatus(orderID, domain.OrderInProgress)
}

func (s *OrderService) EndTrip(orderID string) (domain.RideOrder, error) {
	order, err := s.store.UpdateOrderStatus(orderID, domain.OrderWaitingPayment)
	if err != nil {
		return domain.RideOrder{}, err
	}
	s.store.CreatePayment(orderID, order.FinalAmount)
	return order, nil
}

func (s *OrderService) Cancel(orderID, reason string) (domain.RideOrder, error) {
	order, ok := s.store.GetOrder(orderID)
	if !ok {
		return domain.RideOrder{}, fmt.Errorf("订单不存在")
	}
	if !domain.CanCancelOrder(order.Status) {
		return domain.RideOrder{}, fmt.Errorf("当前状态不允许普通取消")
	}
	return s.store.UpdateOrderStatus(orderID, domain.OrderCanceled)
}

func (s *OrderService) Review(orderID string, score int, content string) domain.Review {
	return s.store.CreateReview(orderID, score, content)
}
