package outbox

import "context"

type Publisher interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
}
