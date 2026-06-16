package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

func (r *RabbitMQ) Consume(ctx context.Context, queueName string, wg *sync.WaitGroup, getMaxRetries func([]byte) int, handler func([]byte) error) error {
	ch, err := r.Conn.Channel()
	if err != nil {
		return fmt.Errorf("open consumer channel: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		_ = ch.Close()
		return fmt.Errorf("set qos: %w", err)
	}

	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
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

				wg.Add(1)
				maxRetries := getMaxRetries(msg.Body)

				var lastErr error
				for attempt := 0; attempt <= maxRetries; attempt++ {
					if attempt > 0 {
						time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
						log.Printf("queue=%s retry %d/%d", queueName, attempt, maxRetries)
					}
					if lastErr = handler(msg.Body); lastErr == nil {
						break
					}
					log.Printf("queue=%s attempt %d failed: %v", queueName, attempt+1, lastErr)
				}
				wg.Done()

				if lastErr != nil {
					_ = msg.Nack(false, false)
				} else {
					_ = msg.Ack(false)
				}
			}
		}
	}()

	return nil
}
