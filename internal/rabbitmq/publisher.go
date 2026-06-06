package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Publish sends a JSON body directly to a named queue and waits for a broker
// ack. Use this for queues that do not need scheduled delivery (SMS, push, etc).
// It is safe to call from multiple goroutines.
func (r *RabbitMQ) Publish(ctx context.Context, queue string, body []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.Channel.PublishWithContext(ctx,
		"",    // default exchange — routes directly to queue by name
		queue, // routing key
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    uuid.NewString(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish to %q: %w", queue, err)
	}

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

// PublishDelayed sends a JSON body to the queue's delayed exchange with an
// x-delay header. The broker holds the message for `delay` before routing it
// to the queue. Pass delay=0 (or any negative value) for immediate delivery.
// It is safe to call from multiple goroutines.
func (r *RabbitMQ) PublishDelayed(ctx context.Context, queue string, body []byte, delay time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delayMs := delay.Milliseconds()
	if delayMs < 0 {
		delayMs = 0
	}

	exchangeName := queue + ".delayed"

	err := r.Channel.PublishWithContext(ctx,
		exchangeName, // route through the delayed exchange
		queue,        // routing key — matches the binding set in DeclareDelayedExchange
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    uuid.NewString(),
			Headers: amqp.Table{
				"x-delay": delayMs, // milliseconds; 0 = deliver immediately
			},
			Body: body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish delayed to %q: %w", exchangeName, err)
	}

	select {
	case confirm, ok := <-r.confirms:
		if !ok {
			return fmt.Errorf("publish delayed to %q: confirm channel closed", exchangeName)
		}
		if !confirm.Ack {
			return fmt.Errorf("publish delayed to %q: broker nack'd message", exchangeName)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("publish delayed to %q: %w", exchangeName, ctx.Err())
	}
}
