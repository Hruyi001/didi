package store

import (
	"testing"

	"didi/backend/internal/domain"
)

func TestUpdateAndGetDriverLocation(t *testing.T) {
	s := NewMemoryStore()
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	location := domain.DriverLocation{DriverID: driver.ID, Lng: 116.397, Lat: 39.908, SpeedKPH: 32}
	updated, err := s.UpdateDriverLocation(driver.ID, location)
	if err != nil {
		t.Fatalf("update location: %v", err)
	}
	if updated.DriverID != driver.ID {
		t.Fatalf("expected driver id %s, got %s", driver.ID, updated.DriverID)
	}
	loaded, ok := s.GetDriverLocation(driver.ID)
	if !ok {
		t.Fatal("expected stored driver location")
	}
	if loaded.Lng != 116.397 || loaded.Lat != 39.908 {
		t.Fatalf("unexpected location %+v", loaded)
	}
}
