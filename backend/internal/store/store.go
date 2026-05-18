package store

import "didi/backend/internal/domain"

type Store interface {
	SeedPassenger(phone string) domain.PassengerProfile
	SeedApprovedDriver(phone, plate string) domain.DriverProfile
	GetPassengerByPhone(phone string) (domain.PassengerProfile, bool)
	GetDriverByPhone(phone string) (domain.DriverProfile, bool)
	ListDrivers() []domain.DriverProfile
	FindOnlineIdleDrivers() []domain.DriverProfile
	SetDriverWorkStatus(driverID string, status domain.DriverWorkStatus) (domain.DriverProfile, error)
	UpdateDriverLocation(driverID string, location domain.DriverLocation) (domain.DriverLocation, error)
	GetDriverLocation(driverID string) (domain.DriverLocation, bool)
	CreateOrder(order domain.RideOrder) domain.RideOrder
	GetOrder(orderID string) (domain.RideOrder, bool)
	ListOrders() []domain.RideOrder
	ListOrdersByPassenger(passengerID string) []domain.RideOrder
	ListOrdersForDriver(driverID string) []domain.RideOrder
	UpdateOrderStatus(orderID string, to domain.OrderStatus) (domain.RideOrder, error)
	AssignDriver(orderID, driverID string) (domain.RideOrder, error)
	CreateDispatchTask(orderID string, candidates int) domain.DispatchTask
	AddDispatchAttempt(attempt domain.DispatchAttempt) domain.DispatchAttempt
	ListDispatchAttempts(orderID string) []domain.DispatchAttempt
	ResetDriverToIdle(driverID string) error
	CreatePayment(orderID string, amount int64) domain.PaymentOrder
	MarkPaymentPaid(orderID string) (domain.PaymentOrder, error)
	CreateReview(orderID string, score int, content string) domain.Review
}
