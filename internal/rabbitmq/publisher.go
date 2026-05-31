package rabbitmq

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Publish sends a JSON body to the named queue and waits for a broker ack.
// It is safe to call from multiple goroutines.
func (r *RabbitMQ) Publish(ctx context.Context, queue string, body []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.Channel.PublishWithContext(ctx,
		"",    // default exchange — routes directly to queue by name
		queue, // routing key
		false, // mandatory: return an error if the queue doesn't exist or can't accept the message
			   // use true to get unroutable messages back in a Return channel instead of getting an error here, but that adds complexity we don't need right now
		false, // immediate (removed in RabbitMQ 3+)
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // survives broker restart
			MessageId:    uuid.NewString(),
			Body:         body,
		},
	)
	//Connection closed, Channel closed, Socket error
	if err != nil {
		return fmt.Errorf("publish to %q: %w", queue, err)
	}

	// Block until the broker acks or nacks, or the caller cancels.
	select {
	case confirm, ok := <-r.confirms:
		if !ok {
			return fmt.Errorf("publish to %q: confirm channel closed", queue)
		}
		if !confirm.Ack {
			return fmt.Errorf("publish to %q: broker nack'd message", queue)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("publish to %q: %w", queue, ctx.Err())
	}
}
