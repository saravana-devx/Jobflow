package rabbitmq

import (
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"jobflow/internal/config"
)

// Connection retry settings. RabbitMQ takes ~20s to boot and its AMQP listener
// on 5672 comes up shortly after the node reports healthy, so the first few
// dials can be refused. Retry with a fixed delay instead of dying immediately.
const (
	connectMaxAttempts = 30
	connectRetryDelay  = 2 * time.Second
)

type RabbitMQ struct {
	Conn     *amqp.Connection
	Channel  *amqp.Channel
	mu       sync.Mutex //only one goroutine can perform the publish + confirmation workflow at a time
	confirms <-chan amqp.Confirmation // tcp socket delivers messages in order, so we can correlate confirms to publishes by waiting for the next confirm after each publish
}

func NewRabbitMQConnection() *RabbitMQ {
	url := config.Get().RabbitMQURL()

	// connect to RabbitMQ, retrying while the broker finishes starting up
	var conn *amqp.Connection
	var err error
	for attempt := 1; attempt <= connectMaxAttempts; attempt++ {
		conn, err = amqp.Dial(url)
		// If there is no error then we have successfully connected to RabbitMQ, so we can break out of the loop
		if err == nil {
			break
		}
		log.Printf("RabbitMQ not ready (attempt %d/%d): %s", attempt, connectMaxAttempts, err)
		time.Sleep(connectRetryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ after %d attempts: %s", connectMaxAttempts, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a RabbitMQ channel: %s", err)
	}

	// Put channel in confirm mode so every Publish gets a broker ack/nack.
	if err := ch.Confirm(false); err != nil {
		log.Fatalf("Failed to enable publisher confirms: %s", err)
	}

	return &RabbitMQ{
		Conn:     conn,
		Channel:  ch,
		confirms: ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}
}

func (r *RabbitMQ) close() {
	if r.Channel != nil {
		if err := r.Channel.Close(); err != nil {
			log.Printf("Error closing RabbitMQ channel: %s", err)
		}
	}
	if r.Conn != nil {
		if err := r.Conn.Close(); err != nil {
			log.Printf("Error closing RabbitMQ connection: %s", err)
		}
	}
}
