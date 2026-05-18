package order

type Order struct {
	ID          string `json:"id"`
	PassengerID string `json:"passengerId"`
	Status      string `json:"status"`
	Pickup      string `json:"pickup"`
	Dropoff     string `json:"dropoff"`
	Version     int64  `json:"version"`
}

type EventPublisher interface {
	PublishOrderCreated(orderID string) error
}

type Service struct {
	repo      *MemoryRepo
	publisher EventPublisher
}

func NewService(repo *MemoryRepo, publisher EventPublisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Create(passengerID, pickup, dropoff string) Order {
	order := s.repo.Create(passengerID, pickup, dropoff)
	if s.publisher != nil {
		_ = s.publisher.PublishOrderCreated(order.ID)
	}
	return order
}
