package user

type PassengerProfile struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	Phone     string `json:"phone"`
	Nickname  string `json:"nickname"`
}

type Service struct{ repo *MemoryRepo }

func NewService(repo *MemoryRepo) *Service { return &Service{repo: repo} }

func (s *Service) EnsurePassenger(accountID, phone string) PassengerProfile {
	return s.repo.EnsurePassenger(accountID, phone)
}
