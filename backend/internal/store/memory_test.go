package store

import (
	"testing"

	"didi/backend/internal/domain"
)

func TestCreateOrderAndUpdateStatus(t *testing.T) {
	s := NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	order := domain.RideOrder{PassengerID: passenger.ID, Pickup: domain.Point{Name: "A"}, Dropoff: domain.Point{Name: "B"}}
	created := s.CreateOrder(order)
	if created.Status != domain.OrderCreated {
		t.Fatalf("expected created status, got %s", created.Status)
	}
	updated, err := s.UpdateOrderStatus(created.ID, domain.OrderDispatching)
	if err != nil {
		t.Fatalf("expected update to pass: %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("expected version 2, got %d", updated.Version)
	}
}

func TestFindOnlineIdleDrivers(t *testing.T) {
	s := NewMemoryStore()
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle)
	drivers := s.FindOnlineIdleDrivers()
	if len(drivers) != 1 || drivers[0].ID != driver.ID {
		t.Fatalf("expected one idle driver")
	}
}
