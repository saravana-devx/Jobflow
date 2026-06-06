package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Consume registers one consumer goroutine on queueName. Each call opens its
// own AMQP channel so Ack/Nack calls never race across concurrent consumers.
// Call Consume N times to run N parallel consumers.
func (r *RabbitMQ) Consume(ctx context.Context, queueName string, maxRetries int, handler func([]byte) error) error {
	ch, err := r.Conn.Channel()
	if err != nil {
		return fmt.Errorf("open consumer channel: %w", err)
	}

	// Prefetch 1: the broker delivers the next message only after the consumer
	// acks the current one. This ensures fair dispatch across multiple consumers.
	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		return fmt.Errorf("set qos: %w", err)
	}

	msgs, err := ch.Consume(
		queueName,
		"",    // consumer tag — broker assigns a unique one
		false, // auto-ack: we ack manually after successful processing
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("start consume on %q: %w", queueName, err)
	}

	go func() {
		defer ch.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Printf("consumer channel closed for queue=%s", queueName)
					return
				}

				var lastErr error
				for attempt := 0; attempt <= maxRetries; attempt++ {
					if attempt > 0 {
						// exponential backoff: 1s, 2s, 4s, …
						time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
						log.Printf("queue=%s retry %d/%d", queueName, attempt, maxRetries)
					}
					if lastErr = handler(msg.Body); lastErr == nil {
						break
					}
					log.Printf("queue=%s attempt %d failed: %v", queueName, attempt+1, lastErr)
				}

				if lastErr != nil {
					// All retries exhausted — dead-letter and move on.
					_ = msg.Nack(false, false)
				} else {
					_ = msg.Ack(false)
				}
			}
		}
	}()

	return nil
}
