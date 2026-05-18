package admin

type DriverSnapshot struct {
	ID         string `json:"id"`
	AuditState string `json:"auditState"`
}

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) ListDrivers() []DriverSnapshot {
	return []DriverSnapshot{{ID: "driver-1", AuditState: "APPROVED"}}
}
