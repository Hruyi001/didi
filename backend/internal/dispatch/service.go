package dispatch

import "fmt"

type Attempt struct {
	OrderID  string `json:"orderId"`
	DriverID string `json:"driverId"`
	Status   string `json:"status"`
	Sequence int    `json:"sequence"`
}

type Publisher interface {
	PublishDispatchAccepted(orderID, driverID string) error
}

type Service struct {
	repo      *MemoryRepo
	publisher Publisher
}

func NewService(repo *MemoryRepo, publisher Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Start(orderID string, candidateDriverIDs []string) (Attempt, error) {
	if len(candidateDriverIDs) == 0 {
		return Attempt{}, fmt.Errorf("no candidates")
	}
	attempt := Attempt{OrderID: orderID, DriverID: candidateDriverIDs[0], Status: "OFFERED", Sequence: 1}
	s.repo.Save(attempt)
	return attempt, nil
}
