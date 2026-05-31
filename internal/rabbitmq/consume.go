package rabbitmq

import (
	"context"
	"log"
	"time"
)

func (r *RabbitMQ) Consume(ctx context.Context, queueName string, maxRetries int, handler func([]byte) error) error {
	// Process one message at a time
	if err := r.Channel.Qos(1, 0, false); err != nil {
		return err
	}

	msgs, err := r.Channel.Consume(
		queueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Println("Message channel closed")
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
					// All retries exhausted — discard so the message doesn't loop forever.
					_ = msg.Nack(false, false)
				} else {
					_ = msg.Ack(false)
				}
			}
		}
	}()

	return nil
}
