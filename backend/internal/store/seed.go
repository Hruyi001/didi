package store

import (
	"fmt"

	"didi/backend/internal/domain"
)

func SeedDemoData(s *MemoryStore) {
	if err := SeedDemoDataForStore(s); err != nil {
		panic(err)
	}
}

func SeedDemoDataForStore(s Store) error {
	s.SeedPassenger("13800000001")
	d1 := s.SeedApprovedDriver("13900000001", "京A12345")
	d2 := s.SeedApprovedDriver("13900000002", "京B67890")
	if _, err := s.SetDriverWorkStatus(d1.ID, domain.DriverOnlineIdle); err != nil {
		return fmt.Errorf("set demo driver 1 online: %w", err)
	}
	if _, err := s.SetDriverWorkStatus(d2.ID, domain.DriverOnlineIdle); err != nil {
		return fmt.Errorf("set demo driver 2 online: %w", err)
	}
	return nil
}
