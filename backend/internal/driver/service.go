package driver

import "fmt"

type Driver struct {
	ID         string `json:"id"`
	Phone      string `json:"phone"`
	AuditState string `json:"auditState"`
	WorkStatus string `json:"workStatus"`
}

type Service struct{ repo *MemoryRepo }

func NewService(repo *MemoryRepo) *Service { return &Service{repo: repo} }

func (s *Service) SetOnline(driverID string) (Driver, error) {
	driver, ok := s.repo.Get(driverID)
	if !ok {
		return Driver{}, fmt.Errorf("driver not found")
	}
	if driver.AuditState != "APPROVED" {
		return Driver{}, fmt.Errorf("driver not approved")
	}
	driver.WorkStatus = "ONLINE_IDLE"
	s.repo.Save(driver)
	return driver, nil
}
