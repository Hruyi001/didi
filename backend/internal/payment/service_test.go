package payment

import "testing"

func TestMarkPaidTransitionsPayment(t *testing.T) {
	repo := NewMemoryRepo()
	svc := NewService(repo, nil)
	created := repo.Create("order-1", 2800)
	paid, err := svc.Pay(created.OrderID)
	if err != nil {
		t.Fatalf("expected pay success: %v", err)
	}
	if paid.Status != "PAID" {
		t.Fatalf("expected PAID, got %s", paid.Status)
	}
}
