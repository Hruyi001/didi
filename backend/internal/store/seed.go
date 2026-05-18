package store

import "didi/backend/internal/domain"

func SeedDemoData(s *MemoryStore) {
	s.SeedPassenger("13800000001")
	d1 := s.SeedApprovedDriver("13900000001", "京A12345")
	d2 := s.SeedApprovedDriver("13900000002", "京B67890")
	s.SetDriverWorkStatus(d1.ID, domain.DriverOnlineIdle)
	s.SetDriverWorkStatus(d2.ID, domain.DriverOnlineIdle)
}
