// File: internal/sse/subscriber.go
package sse

import (
	"context"
	"strings"

	"jobflow/internal/redis"
)

type Subscriber struct {
	rdb     *redis.Redis
	manager *ClientManager
}

func NewSubscriber(rdb *redis.Redis, manager *ClientManager) *Subscriber {
	return &Subscriber{rdb: rdb, manager: manager}
}

// Start subscribes to the given Redis channels and relays every message to all
// connected SSE clients. Blocks until ctx is cancelled — run in a goroutine.
func (s *Subscriber) Start(ctx context.Context, patterns ...string) {
	pubsub := s.rdb.PSubscribe(ctx, patterns...)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			// channel format: "user:{userID}:jobs" — extract the middle segment
			parts := strings.SplitN(msg.Channel, ":", 3)
			if len(parts) == 3 {
				s.manager.SendToClient(parts[1], msg.Payload)
			}
		case <-ctx.Done():
			return
		}
	}
}
