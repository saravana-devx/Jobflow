package main

import (
    "log"
    "pulseDashboard/internal/config"
    "pulseDashboard/internal/rabbitmq"
    "pulseDashboard/internal/worker"
)

func main() {
    if err := config.Load(); err != nil {
        log.Fatalf("config load failed: %v", err)
    }

    mq := rabbitmq.NewRabbitMQConnection()
    w := worker.New(mq)
    w.Start()
}