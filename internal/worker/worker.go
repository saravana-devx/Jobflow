package worker

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"pulseDashboard/internal/rabbitmq"
)

type Worker struct {
	mq *rabbitmq.RabbitMQ
}

func New(mq *rabbitmq.RabbitMQ) *Worker {
	return &Worker{mq: mq}
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

	err := w.mq.Consume(ctx, rabbitmq.QueueJobs, 3, func(body []byte) error {
		wg.Add(1)
		defer wg.Done()
		return handleJob(body)
	})
	if err != nil {
		log.Fatalf("failed to start consumer: %v", err)
	}

	log.Println("Worker started, waiting for jobs...")
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
