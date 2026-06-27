// File: internal/health/handler.go
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"jobflow/internal/rabbitmq"
	"jobflow/internal/redis"
)

const (
	statusUp   = "up"
	statusDown = "down"
)

type serviceStatus struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Handler struct {
	db  *gorm.DB
	rdb *redis.Redis
	mq  *rabbitmq.RabbitMQ
}

func NewHandler(db *gorm.DB, rdb *redis.Redis, mq *rabbitmq.RabbitMQ) *Handler {
	return &Handler{db: db, rdb: rdb, mq: mq}
}

func (h *Handler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	results := map[string]serviceStatus{
		"postgres": h.checkPostgres(ctx),
		"redis":    h.checkRedis(ctx),
		"rabbitmq": h.checkRabbitMQ(),
	}

	overall := statusUp
	code := http.StatusOK
	for _, s := range results {
		if s.Status == statusDown {
			overall = statusDown
			code = http.StatusServiceUnavailable
			break
		}
	}

	c.JSON(code, gin.H{
		"status":   overall,
		"services": results,
	})
}

func (h *Handler) checkPostgres(ctx context.Context) serviceStatus {
	start := time.Now()
	sqlDB, err := h.db.DB()
	if err != nil {
		return serviceStatus{Status: statusDown, Error: err.Error()}
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return serviceStatus{Status: statusDown, Error: err.Error()}
	}
	return serviceStatus{Status: statusUp, Latency: time.Since(start).String()}
}

func (h *Handler) checkRedis(ctx context.Context) serviceStatus {
	start := time.Now()
	if err := h.rdb.Client.Ping(ctx).Err(); err != nil {
		return serviceStatus{Status: statusDown, Error: err.Error()}
	}
	return serviceStatus{Status: statusUp, Latency: time.Since(start).String()}
}

// checkRabbitMQ probes the broker. amqp091-go's connection/channel calls aren't
// context-aware, so the route's timeout doesn't bound this directly; we keep the
// probe cheap instead. Opening a channel is a round-trip to the broker, so it
// confirms the broker actually answers — not just that our local socket is open.
func (h *Handler) checkRabbitMQ() serviceStatus {
	start := time.Now()
	if h.mq == nil || h.mq.Conn == nil || h.mq.Conn.IsClosed() {
		return serviceStatus{Status: statusDown, Error: "connection is closed"}
	}
	ch, err := h.mq.Conn.Channel()
	if err != nil {
		return serviceStatus{Status: statusDown, Error: err.Error()}
	}
	_ = ch.Close()
	return serviceStatus{Status: statusUp, Latency: time.Since(start).String()}
}
