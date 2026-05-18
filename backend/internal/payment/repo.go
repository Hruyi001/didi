package payment

type MemoryRepo struct {
	payments map[string]Payment
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{payments: map[string]Payment{}}
}

func (r *MemoryRepo) Create(orderID string, amount int64) Payment {
	payment := Payment{OrderID: orderID, Amount: amount, Status: "UNPAID"}
	r.payments[orderID] = payment
	return payment
}

func (r *MemoryRepo) Get(orderID string) (Payment, bool) {
	payment, ok := r.payments[orderID]
	return payment, ok
}

func (r *MemoryRepo) Save(payment Payment) {
	r.payments[payment.OrderID] = payment
}
