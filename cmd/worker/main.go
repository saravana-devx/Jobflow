package main

import (
	"log"
	"jobflow/internal/config"
	"jobflow/internal/database"
	"jobflow/internal/rabbitmq"
	"jobflow/internal/redis"
	"jobflow/internal/worker"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	mq := rabbitmq.NewRabbitMQConnection()
	if err := mq.InitializeQueues(rabbitmq.AppQueues); err != nil {
		log.Fatalf("failed to initialize queues: %v", err)
	}
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	rdb := redis.NewRedis()
	w := worker.New(mq, db, rdb)
	w.Start()
}
