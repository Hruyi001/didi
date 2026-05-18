package admin

import "testing"

func TestListDriversReturnsSeededSnapshot(t *testing.T) {
	svc := NewService()
	drivers := svc.ListDrivers()
	if len(drivers) == 0 {
		t.Fatal("expected seeded driver snapshot")
	}
	if drivers[0].AuditState == "" {
		t.Fatal("expected audit state")
	}
}
