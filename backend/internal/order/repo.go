package order

import "github.com/google/uuid"

type MemoryRepo struct {
	orders map[string]Order
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{orders: map[string]Order{}}
}

func (r *MemoryRepo) Create(passengerID, pickup, dropoff string) Order {
	order := Order{
		ID:          uuid.NewString(),
		PassengerID: passengerID,
		Status:      "CREATED",
		Pickup:      pickup,
		Dropoff:     dropoff,
		Version:     1,
	}
	r.orders[order.ID] = order
	return order
}
