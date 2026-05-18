package location

import "testing"

func TestNearbyDriversAreSortedByDistance(t *testing.T) {
	repo := NewMemoryRepo()
	repo.Upsert("d1", 116.390, 39.900)
	repo.Upsert("d2", 116.401, 39.901)
	svc := NewService(repo)
	drivers := svc.Nearby(116.398, 39.900, 2)
	if len(drivers) != 2 {
		t.Fatalf("expected 2 drivers, got %d", len(drivers))
	}
}
