package kafka

import "testing"

func TestTopicNamesAreStable(t *testing.T) {
	if OrderCreatedTopic != "ride.order.created" {
		t.Fatalf("unexpected order-created topic: %q", OrderCreatedTopic)
	}
	if PaymentPaidTopic != "ride.payment.paid" {
		t.Fatalf("unexpected payment-paid topic: %q", PaymentPaidTopic)
	}
}
