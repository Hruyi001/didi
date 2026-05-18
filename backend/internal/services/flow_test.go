package services

import (
	"testing"

	"didi/backend/internal/domain"
	"didi/backend/internal/store"
)

func TestRideFlowFromCreateToPayment(t *testing.T) {
	s := store.NewMemoryStore()
	passenger := s.SeedPassenger("13800000001")
	driver := s.SeedApprovedDriver("13900000001", "京A12345")
	if _, err := s.SetDriverWorkStatus(driver.ID, domain.DriverOnlineIdle); err != nil {
		t.Fatalf("driver online: %v", err)
	}

	orders := NewOrderService(s)
	dispatch := NewDispatchService(s)
	payments := NewPaymentService(s)

	order := orders.CreateRide(passenger.ID, domain.Point{Name: "上车点", Lng: 116.397, Lat: 39.908}, domain.Point{Name: "下车点", Lng: 116.407, Lat: 39.918})
	if order.Status != domain.OrderCreated {
		t.Fatalf("expected CREATED, got %s", order.Status)
	}
	attempt, err := dispatch.Dispatch(order.ID)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if attempt.DriverID != driver.ID {
		t.Fatalf("expected dispatch to seed driver")
	}
	order, err = dispatch.Accept(order.ID, driver.ID)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if order.Status != domain.OrderWaitingPickup {
		t.Fatalf("expected WAITING_PICKUP, got %s", order.Status)
	}
	if order, err = orders.Arrive(order.ID); err != nil || order.Status != domain.OrderDriverArrived {
		t.Fatalf("arrive status=%s err=%v", order.Status, err)
	}
	if order, err = orders.StartTrip(order.ID); err != nil || order.Status != domain.OrderInProgress {
		t.Fatalf("start status=%s err=%v", order.Status, err)
	}
	if order, err = orders.EndTrip(order.ID); err != nil || order.Status != domain.OrderWaitingPayment {
		t.Fatalf("end status=%s err=%v", order.Status, err)
	}
	_, order, err = payments.Pay(order.ID)
	if err != nil {
		t.Fatalf("pay: %v", err)
	}
	if order.Status != domain.OrderCompleted {
		t.Fatalf("expected COMPLETED, got %s", order.Status)
	}
}
