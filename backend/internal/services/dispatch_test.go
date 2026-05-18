package services

import (
	"testing"
	"time"

	"didi/backend/internal/domain"
	"didi/backend/internal/store"
)

func TestRejectRedispatchesToNextDriver(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driverA := s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京A54321")
	if _, err := s.SetDriverWorkStatus(driverA.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver A online: %v", err)
	}
	if _, err := s.SetDriverWorkStatus(driverB.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver B online: %v", err)
	}

	orders := NewOrderService(s)
	dispatch := NewDispatchService(s)
	order := orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})

	attempt, err := dispatch.Dispatch(order.ID)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if attempt.DriverID != driverA.ID {
		t.Fatalf("expected first offer to driver A, got %s", attempt.DriverID)
	}

	nextAttempt, err := dispatch.Reject(order.ID, driverA.ID, "司机拒单")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if nextAttempt.DriverID != driverB.ID {
		t.Fatalf("expected redispatch to driver B, got %s", nextAttempt.DriverID)
	}
	if nextAttempt.Status != domain.DispatchOffered {
		t.Fatalf("expected offered status, got %s", nextAttempt.Status)
	}

	updatedOrder, ok := s.GetOrder(order.ID)
	if !ok {
		t.Fatal("expected order to exist")
	}
	if updatedOrder.Status != domain.OrderDispatching {
		t.Fatalf("expected order to remain DISPATCHING, got %s", updatedOrder.Status)
	}
}

func TestRejectFailsOrderAfterMaxAttempts(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driverA := s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京A54321")
	driverC := s.SeedApprovedDriver("13900000003", "京A67890")
	for _, driverID := range []string{driverA.ID, driverB.ID, driverC.ID} {
		if _, err := s.SetDriverWorkStatus(driverID, domain.DriverOnlineIdle); err != nil {
			t.Fatalf("set driver online: %v", err)
		}
	}

	orders := NewOrderService(s)
	dispatch := NewDispatchService(s)
	order := orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})

	firstAttempt, err := dispatch.Dispatch(order.ID)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if _, err := dispatch.Reject(order.ID, firstAttempt.DriverID, "司机拒单"); err != nil {
		t.Fatalf("first reject: %v", err)
	}
	secondAttempt, err := dispatch.Reject(order.ID, driverB.ID, "司机拒单")
	if err != nil {
		t.Fatalf("second reject: %v", err)
	}
	if _, err := dispatch.Reject(order.ID, secondAttempt.DriverID, "司机拒单"); err == nil {
		t.Fatal("expected final reject to fail order")
	}

	updatedOrder, ok := s.GetOrder(order.ID)
	if !ok {
		t.Fatal("expected order to exist")
	}
	if updatedOrder.Status != domain.OrderDispatchFailed {
		t.Fatalf("expected DISPATCH_FAILED, got %s", updatedOrder.Status)
	}
}

func TestTimeoutRedispatchesToNextDriver(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driverA := s.SeedApprovedDriver("13900000001", "京A12345")
	driverB := s.SeedApprovedDriver("13900000002", "京A54321")
	if _, err := s.SetDriverWorkStatus(driverA.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver A online: %v", err)
	}
	if _, err := s.SetDriverWorkStatus(driverB.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver B online: %v", err)
	}

	orders := NewOrderService(s)
	dispatch := NewDispatchService(s)
	order := orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})

	if _, err := dispatch.Dispatch(order.ID); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if err := dispatch.ProcessTimeouts(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatalf("process timeouts: %v", err)
	}

	attempts := s.ListDispatchAttempts(order.ID)
	if len(attempts) != 3 {
		t.Fatalf("expected timeout record and redispatch, got %d attempts", len(attempts))
	}
	if attempts[1].Status != domain.DispatchTimeout {
		t.Fatalf("expected timeout attempt, got %s", attempts[1].Status)
	}
	if attempts[2].DriverID != driverB.ID {
		t.Fatalf("expected redispatch to driver B, got %s", attempts[2].DriverID)
	}
}

func TestAcceptRecordsAcceptedDispatchAttempt(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	if _, err := s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("set driver online: %v", err)
	}

	orders := NewOrderService(s)
	dispatch := NewDispatchService(s)
	order := orders.CreateRide(passenger.ID, domain.Point{Name: "A"}, domain.Point{Name: "B"})

	firstAttempt, err := dispatch.Dispatch(order.ID)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if _, err := dispatch.Accept(order.ID, driver.ID); err != nil {
		t.Fatalf("accept: %v", err)
	}

	attempts := s.ListDispatchAttempts(order.ID)
	if len(attempts) != 2 {
		t.Fatalf("expected offered and accepted attempts, got %d", len(attempts))
	}
	if attempts[0].Status != domain.DispatchOffered {
		t.Fatalf("expected first attempt offered, got %s", attempts[0].Status)
	}
	if attempts[0].DriverID != firstAttempt.DriverID {
		t.Fatalf("expected first attempt driver %s, got %s", firstAttempt.DriverID, attempts[0].DriverID)
	}
	if attempts[1].Status != domain.DispatchAccepted {
		t.Fatalf("expected accepted attempt, got %s", attempts[1].Status)
	}
	if attempts[1].DriverID != driver.ID {
		t.Fatalf("expected accepted attempt for driver %s, got %s", driver.ID, attempts[1].DriverID)
	}
}
