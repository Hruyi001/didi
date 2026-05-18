package domain

import "fmt"

var orderTransitions = map[OrderStatus]map[OrderStatus]bool{
	OrderCreated: {
		OrderDispatching:    true,
		OrderCanceled:       true,
		OrderDispatchFailed: true,
	},
	OrderDispatching: {
		OrderWaitingPickup:  true,
		OrderCanceled:       true,
		OrderDispatchFailed: true,
	},
	OrderWaitingPickup: {
		OrderDriverArrived: true,
		OrderCanceled:      true,
	},
	OrderDriverArrived: {
		OrderInProgress: true,
		OrderCanceled:   true,
	},
	OrderInProgress: {
		OrderWaitingPayment: true,
	},
	OrderWaitingPayment: {
		OrderCompleted: true,
	},
}

func CanTransitionOrder(from, to OrderStatus) error {
	if orderTransitions[from][to] {
		return nil
	}
	return fmt.Errorf("invalid order transition %s -> %s", from, to)
}

func CanCancelOrder(status OrderStatus) bool {
	switch status {
	case OrderCreated, OrderDispatching, OrderWaitingPickup, OrderDriverArrived:
		return true
	default:
		return false
	}
}
