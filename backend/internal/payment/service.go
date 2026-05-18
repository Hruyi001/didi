package payment

import "fmt"

type Payment struct {
	OrderID string `json:"orderId"`
	Amount  int64  `json:"amount"`
	Status  string `json:"status"`
}

type Publisher interface {
	PublishPaymentPaid(orderID string) error
}

type Service struct {
	repo      *MemoryRepo
	publisher Publisher
}

func NewService(repo *MemoryRepo, publisher Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Pay(orderID string) (Payment, error) {
	payment, ok := s.repo.Get(orderID)
	if !ok {
		return Payment{}, fmt.Errorf("payment not found")
	}
	payment.Status = "PAID"
	s.repo.Save(payment)
	if s.publisher != nil {
		_ = s.publisher.PublishPaymentPaid(orderID)
	}
	return payment, nil
}
