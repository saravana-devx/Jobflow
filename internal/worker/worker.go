package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gorm.io/gorm"
	"jobflow/internal/rabbitmq"
	redisx "jobflow/internal/redis"
)

const numConsumers = 5
const defaultMaxRetries = 3

type Worker struct {
	mq       *rabbitmq.RabbitMQ
	repo     *WorkerRepository
	rdb      *redisx.Redis
	workerID string
}

func New(mq *rabbitmq.RabbitMQ, db *gorm.DB, rdb *redisx.Redis) *Worker {
	hostname, _ := os.Hostname()
	workerID := fmt.Sprintf("%s-%d", hostname, os.Getpid())
	return &Worker{mq: mq, repo: NewWorkerRepository(db), rdb: rdb, workerID: workerID}
}

func (w *Worker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutdown signal received")
		cancel()
	}()

	for range numConsumers {
		if err := w.mq.Consume(ctx, rabbitmq.QueueJobs, jobMaxRetries, func(body []byte) error {
			wg.Add(1)
			defer wg.Done()
			return w.handleJob(body)
		}); err != nil {
			log.Fatalf("failed to start consumer: %v", err)
		}
	}

	log.Printf("Worker started: id=%s consumers=%d", w.workerID, numConsumers)
	<-ctx.Done()
	w.drain(&wg)
}

func (w *Worker) drain(wg *sync.WaitGroup) {
	log.Println("Draining in-flight jobs...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All jobs completed cleanly")
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout reached, forcing exit")
	}

	log.Println("Worker stopped")
}
