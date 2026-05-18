package dispatch

import "testing"

func TestDispatchSelectsFirstCandidate(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo, nil)
	attempt, err := svc.Start("order-1", []string{"driver-1", "driver-2"})
	if err != nil {
		t.Fatalf("expected dispatch success: %v", err)
	}
	if attempt.DriverID != "driver-1" {
		t.Fatalf("expected first driver, got %s", attempt.DriverID)
	}
}
