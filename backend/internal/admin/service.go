package admin

type DriverSnapshot struct {
	ID         string `json:"id"`
	Phone      string `json:"phone"`
	AuditState string `json:"auditState"`
	WorkStatus string `json:"workStatus"`
	PlateNo    string `json:"plateNo"`
}

type Service struct {
	drivers DriverRepository
}

func NewService(drivers DriverRepository) *Service {
	return &Service{drivers: drivers}
}

func (s *Service) ListDrivers() ([]DriverSnapshot, error) {
	return s.drivers.ListDrivers()
}
