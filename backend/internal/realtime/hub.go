package realtime

import "sync"

type Hub struct {
	mu   sync.RWMutex
	subs map[string][]chan []byte
}

func NewHub() *Hub { return &Hub{subs: map[string][]chan []byte{}} }

func (h *Hub) Subscribe(key string) chan []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan []byte, 4)
	h.subs[key] = append(h.subs[key], ch)
	return ch
}

func (h *Hub) Publish(key string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs[key] {
		ch <- payload
	}
}
