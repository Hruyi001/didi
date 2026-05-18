package realtime

import "testing"

func TestHubBroadcastsToSubscribers(t *testing.T) {
	hub := NewHub()
	ch := hub.Subscribe("passenger:1")
	hub.Publish("passenger:1", []byte(`{"type":"ORDER_UPDATED"}`))
	msg := <-ch
	if string(msg) == "" {
		t.Fatal("expected message payload")
	}
}
