package order

import "testing"

func TestCreateOrderStartsInCreatedState(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo, nil)
	order := svc.Create("passenger-1", "pickup", "dropoff")
	if order.Status != "CREATED" {
		t.Fatalf("expected CREATED, got %s", order.Status)
	}
}
