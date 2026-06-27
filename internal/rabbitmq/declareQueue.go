package rabbitmq

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// QueueConfig holds the declaration options for a single queue.
type QueueConfig struct {
	Name                 string
	Durable              bool
	AutoDelete           bool   // delete when the last consumer disconnects
	Exclusive            bool   // only the connection that created the queue can use it
	Type                 string // amqp.QueueTypeQuorum | amqp.QueueTypeClassic | amqp.QueueTypeStream
	DLXName              string // dead-letter exchange; empty = default exchange (route by queue name)
	DeadLetterRoutingKey string // dead-letter target queue name; empty = no dead-lettering
	MessageTTL           int64  // per-message TTL in milliseconds; 0 = no TTL
	UseDelayedExchange   bool   // declare x-delayed-message exchange and bind this queue to it
}

// DefaultQueueConfig returns a durable quorum queue (replicated, survives node loss)
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
	if cfg.DeadLetterRoutingKey != "" {
		// route dead-letters via DLXName (empty = default exchange) to the DLQ
		args["x-dead-letter-exchange"] = cfg.DLXName
		args["x-dead-letter-routing-key"] = cfg.DeadLetterRoutingKey
	} else if cfg.DLXName != "" {
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
		false,
		args,
	)
	if err != nil {
		return amqp.Queue{}, fmt.Errorf("declare queue %q: %w", cfg.Name, err)
	}

	log.Printf("queue declared: name=%s type=%s durable=%v", q.Name, cfg.Type, cfg.Durable)
	return q, nil
}

// DeclareDelayedExchange sets up an x-delayed-message exchange bound to queueName.
// publish with an x-delay header (ms) and the broker holds the message that long.
func (r *RabbitMQ) DeclareDelayedExchange(queueName string) error {
	exchangeName := queueName + ".delayed"

	err := r.Channel.ExchangeDeclare(
		exchangeName,
		"x-delayed-message",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		amqp.Table{
			"x-delayed-type": "direct",
		},
	)
	if err != nil {
		return fmt.Errorf("declare delayed exchange %q: %w", exchangeName, err)
	}

	// bind the queue so routed messages land in it
	if err := r.Channel.QueueBind(queueName, queueName, exchangeName, false, nil); err != nil {
		return fmt.Errorf("bind queue %q to exchange %q: %w", queueName, exchangeName, err)
	}

	log.Printf("delayed exchange declared: exchange=%s bound to queue=%s", exchangeName, queueName)
	return nil
}

// InitializeQueues declares every queue in cfgs (and its delayed exchange if set),
// stopping on the first error.
func (r *RabbitMQ) InitializeQueues(cfgs []QueueConfig) error {
	for _, cfg := range cfgs {
		if _, err := r.DeclareQueue(cfg); err != nil {
			return err
		}
		if cfg.UseDelayedExchange {
			if err := r.DeclareDelayedExchange(cfg.Name); err != nil {
				return err
			}
		}
	}
	return nil
}
