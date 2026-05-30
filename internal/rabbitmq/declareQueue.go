package rabbitmq

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// QueueConfig holds the declaration options for a single queue.
type QueueConfig struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	Type       string // amqp.QueueTypeQuorum | amqp.QueueTypeClassic | amqp.QueueTypeStream
	DLXName    string // dead-letter exchange; empty = disabled
	MessageTTL int64  // per-message TTL in milliseconds; 0 = no TTL
}

// DefaultQueueConfig returns a durable quorum queue config — the right default
// for production workloads (Raft-replicated, survives node failures).
func DefaultQueueConfig(name string) QueueConfig {
	return QueueConfig{
		Name:    name,
		Durable: true,
		Type:    amqp.QueueTypeQuorum,
	}
}

// DeclareQueue declares a queue from the given config.
// It is idempotent: re-declaring with the same parameters is safe.
func (r *RabbitMQ) DeclareQueue(cfg QueueConfig) (amqp.Queue, error) {
	args := amqp.Table{
		amqp.QueueTypeArg: cfg.Type,
	}
	if cfg.DLXName != "" {
		args["x-dead-letter-exchange"] = cfg.DLXName
	}
	if cfg.MessageTTL > 0 {
		args["x-message-ttl"] = cfg.MessageTTL
	}

	q, err := r.Channel.QueueDeclare(
		cfg.Name,
		cfg.Durable,
		cfg.AutoDelete,
		cfg.Exclusive,
		false, // no-wait: always wait for the broker's confirm
		args,
	)
	if err != nil {
		return amqp.Queue{}, fmt.Errorf("declare queue %q: %w", cfg.Name, err)
	}

	log.Printf("queue declared: name=%s type=%s durable=%v", q.Name, cfg.Type, cfg.Durable)
	return q, nil
}

// InitializeQueues declares every queue in cfgs, stopping on the first error.
func (r *RabbitMQ) InitializeQueues(cfgs []QueueConfig) error {
	for _, cfg := range cfgs {
		if _, err := r.DeclareQueue(cfg); err != nil {
			return err
		}
	}
	return nil
}
