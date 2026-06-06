package redis

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

// Publish sends payload to the named Redis pub/sub channel.
// Returns an error if the broker is unreachable or the command fails.
func (r *Redis) Publish(ctx context.Context, channel string, payload []byte) error {
	if err := r.Client.Publish(ctx, channel, payload).Err(); err != nil {
		return fmt.Errorf("redis publish to %q: %w", channel, err)
	}
	return nil
}

// PSubscribe returns a PubSub handle subscribed to the given glob patterns.
// The caller is responsible for closing the handle when done.
func (r *Redis) PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub {
	return r.Client.PSubscribe(ctx, patterns...)
}

