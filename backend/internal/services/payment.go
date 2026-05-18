package services

import (
	"fmt"

	"didi/backend/internal/domain"
	"didi/backend/internal/store"
)

type PaymentService struct{ store store.Store }

func NewPaymentService(store store.Store) *PaymentService { return &PaymentService{store: store} }

func (s *PaymentService) Pay(orderID string) (domain.PaymentOrder, domain.RideOrder, error) {
	order, ok := s.store.GetOrder(orderID)
	if !ok {
		return domain.PaymentOrder{}, domain.RideOrder{}, fmt.Errorf("订单不存在")
	}
	if order.Status != domain.OrderWaitingPayment {
		return domain.PaymentOrder{}, domain.RideOrder{}, fmt.Errorf("订单不是待支付状态")
	}
	payment, err := s.store.MarkPaymentPaid(orderID)
	if err != nil {
		return domain.PaymentOrder{}, domain.RideOrder{}, err
	}
	completed, err := s.store.UpdateOrderStatus(orderID, domain.OrderCompleted)
	return payment, completed, err
}
