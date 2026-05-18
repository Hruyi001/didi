package domain

import "testing"

func TestCanTransitionOrderHappyPath(t *testing.T) {
	path := []OrderStatus{
		OrderCreated,
		OrderDispatching,
		OrderWaitingPickup,
		OrderDriverArrived,
		OrderInProgress,
		OrderWaitingPayment,
		OrderCompleted,
	}
	for i := 0; i < len(path)-1; i++ {
		if err := CanTransitionOrder(path[i], path[i+1]); err != nil {
			t.Fatalf("expected %s -> %s to be valid: %v", path[i], path[i+1], err)
		}
	}
}

func TestRejectInvalidOrderTransition(t *testing.T) {
	if err := CanTransitionOrder(OrderCompleted, OrderInProgress); err == nil {
		t.Fatal("expected completed order not to return to in-progress")
	}
}

func TestCanCancelBeforeTripStarts(t *testing.T) {
	allowed := []OrderStatus{OrderCreated, OrderDispatching, OrderWaitingPickup, OrderDriverArrived}
	for _, status := range allowed {
		if !CanCancelOrder(status) {
			t.Fatalf("expected %s to be cancelable", status)
		}
	}
	if CanCancelOrder(OrderInProgress) {
		t.Fatal("expected in-progress order not to be normally cancelable")
	}
}
