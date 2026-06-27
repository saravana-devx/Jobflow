package rabbitmq

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"jobflow/internal/config"
)

// rabbitmq takes ~20s to boot, so the first few dials can be refused — retry
// instead of dying.
const (
	connectMaxAttempts = 30
	connectRetryDelay  = 2 * time.Second

	// backoff for an existing connection that drops at runtime (broker restart etc.)
	reconnectBaseDelay = 1 * time.Second
	reconnectMaxDelay  = 30 * time.Second
)

type RabbitMQ struct {
	Conn        *amqp.Connection
	Channel     *amqp.Channel
	mu          sync.Mutex               // guards the fields below across publish/reconnect
	confirms    <-chan amqp.Confirmation // publisher confirms, one per publish (in order)
	notifyClose chan *amqp.Error         // broker signals here when the connection closes
	url         string                   // dialled on (re)connect
	closed      atomic.Bool              // set by close() so the watcher stops redialling
}

func NewRabbitMQConnection() *RabbitMQ {
	r := &RabbitMQ{url: config.Get().RabbitMQURL()}

	// Connect, retrying while the broker finishes starting up.
	var err error
	for attempt := 1; attempt <= connectMaxAttempts; attempt++ {
		if err = r.connect(); err == nil {
			break
		}
		log.Printf("RabbitMQ not ready (attempt %d/%d): %s", attempt, connectMaxAttempts, err)
		time.Sleep(connectRetryDelay)
	}
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ after %d attempts: %s", connectMaxAttempts, err)
	}

	go r.watchReconnect()
	return r
}

// connect does one dial: open connection + channel, turn on publisher confirms,
// and swap the live handles in under the lock.
func (r *RabbitMQ) connect() error {
	conn, err := amqp.Dial(r.url)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	// Put channel in confirm mode so every Publish gets a broker ack/nack.
	if err := ch.Confirm(false); err != nil {
		_ = conn.Close()
		return fmt.Errorf("enable confirms: %w", err)
	}

	r.mu.Lock()
	r.Conn = conn
	r.Channel = ch
	r.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	r.notifyClose = conn.NotifyClose(make(chan *amqp.Error, 1))
	r.mu.Unlock()
	return nil
}

// watchReconnect waits for the connection to close and, unless we closed it
// ourselves, redials with backoff and re-declares the topology.
func (r *RabbitMQ) watchReconnect() {
	for {
		r.mu.Lock()
		notify := r.notifyClose
		r.mu.Unlock()

		closeErr := <-notify
		if r.closed.Load() {
			return // we initiated the close; stop watching
		}
		log.Printf("RabbitMQ connection lost: %v; reconnecting...", closeErr)

		delay := reconnectBaseDelay
		for {
			if r.closed.Load() {
				return
			}
			if err := r.connect(); err == nil {
				break
			} else {
				log.Printf("RabbitMQ reconnect failed: %s; retrying in %s", err, delay)
				time.Sleep(delay)
				if delay *= 2; delay > reconnectMaxDelay {
					delay = reconnectMaxDelay
				}
			}
		}

		// a fresh broker (recreated container) has no queues, so re-declare them
		if err := r.InitializeQueues(AppQueues); err != nil {
			log.Printf("RabbitMQ topology re-declare failed after reconnect: %s", err)
		} else {
			log.Println("RabbitMQ reconnected and topology re-declared")
		}
	}
}

func (r *RabbitMQ) close() {
	r.closed.Store(true)

	r.mu.Lock()
	defer r.mu.Unlock()

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
