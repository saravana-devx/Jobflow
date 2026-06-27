package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// how long to wait before reopening a consumer channel after it drops
const consumeRetryDelay = 2 * time.Second

// Consume starts one consumer goroutine for queueName and returns; it runs until
// ctx is cancelled. wg.Add(1) happens here, before the goroutine starts, so a
// drain's wg.Wait() can't see a zero count and return while this consumer is live.
func (r *RabbitMQ) Consume(ctx context.Context, queueName string, wg *sync.WaitGroup, getMaxRetries func([]byte) int, handler func([]byte) error) error {
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.consumeLoop(ctx, queueName, getMaxRetries, handler)
	}()
	return nil
}

// consumeLoop keeps the consumer alive across drops: when consumeOnce returns
// (channel closed / reconnect), it waits and opens a fresh channel, until ctx ends.
func (r *RabbitMQ) consumeLoop(ctx context.Context, queueName string, getMaxRetries func([]byte) int, handler func([]byte) error) {
	for {
		if err := r.consumeOnce(ctx, queueName, getMaxRetries, handler); err != nil {
			log.Printf("queue=%s consume error: %v", queueName, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(consumeRetryDelay):
		}
	}
}

func (r *RabbitMQ) consumeOnce(ctx context.Context, queueName string, getMaxRetries func([]byte) int, handler func([]byte) error) error {
	// grab the current conn under lock so we use the latest one after a reconnect
	r.mu.Lock()
	conn := r.Conn
	r.mu.Unlock()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open consumer channel: %w", err)
	}
	defer ch.Close()

	if err := ch.Qos(1, 0, false); err != nil {
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
		return fmt.Errorf("start consume on %q: %w", queueName, err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("consumer channel closed for queue=%s", queueName)
			}

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

			// out of retries: Nack(requeue=false) so the broker sends it to
			// jobs.dlq instead of dropping it
			if lastErr != nil {
				_ = msg.Nack(false, false)
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}
