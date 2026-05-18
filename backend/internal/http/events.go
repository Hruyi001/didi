package http

import "sync"

type RealtimeEvent struct {
	Type     string  `json:"type"`
	OrderID  string  `json:"orderId,omitempty"`
	DriverID string  `json:"driverId,omitempty"`
	Status   string  `json:"status,omitempty"`
	Lng      float64 `json:"lng,omitempty"`
	Lat      float64 `json:"lat,omitempty"`
	SpeedKPH int     `json:"speedKph,omitempty"`
	Message  string  `json:"message,omitempty"`
}

type eventSubscription struct {
	ch     chan RealtimeEvent
	filter func(RealtimeEvent) bool
}

type EventHub struct {
	mu          sync.RWMutex
	subscribers map[chan RealtimeEvent]eventSubscription
}

func NewEventHub() *EventHub {
	return &EventHub{subscribers: map[chan RealtimeEvent]eventSubscription{}}
}

func (h *EventHub) Subscribe() chan RealtimeEvent {
	return h.SubscribeFiltered(nil)
}

func (h *EventHub) SubscribeFiltered(filter func(RealtimeEvent) bool) chan RealtimeEvent {
	ch := make(chan RealtimeEvent, 8)
	h.mu.Lock()
	h.subscribers[ch] = eventSubscription{ch: ch, filter: filter}
	h.mu.Unlock()
	return ch
}

func (h *EventHub) Unsubscribe(ch chan RealtimeEvent) {
	h.mu.Lock()
	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
	}
	h.mu.Unlock()
}

func (h *EventHub) Publish(event RealtimeEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, subscriber := range h.subscribers {
		if subscriber.filter != nil && !subscriber.filter(event) {
			continue
		}
		select {
		case subscriber.ch <- event:
		default:
		}
	}
}
