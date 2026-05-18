package realtime

type Consumer struct {
	hub *Hub
}

func NewConsumer(hub *Hub) *Consumer {
	return &Consumer{hub: hub}
}
