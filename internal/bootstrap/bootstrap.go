package bootstrap

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"pulseDashboard/internal/auth"
	"pulseDashboard/internal/database"
	"pulseDashboard/internal/jobs"
	"pulseDashboard/internal/rabbitmq"
	"pulseDashboard/internal/redis"
	"pulseDashboard/internal/routes"
)

type App struct {
	Router   *gin.Engine
	DB       *gorm.DB
	Redis    *redis.Redis
	RabbitMQ *rabbitmq.RabbitMQ
}

func New() (*App, error) {
	db, err := database.ConnectDB()
	if err != nil {
		return nil, err
	}
	rdb := redis.NewRedis()

	mq := rabbitmq.NewRabbitMQConnection()
	if err := mq.InitializeQueues(rabbitmq.AppQueues); err != nil {
		return nil, err
	}

	userRepo := auth.NewUserRepository(db)
	jtiStore := auth.NewJTIStore(rdb)
	authService := auth.NewService(userRepo, jtiStore)
	authHandler := auth.NewHandler(authService)

	jobsRepo := jobs.NewJobRepository(db)
	jobsService := jobs.NewService(jobsRepo, mq)
	jobsHandler := jobs.NewHandler(jobsService)

	router := gin.Default()
	// No reverse proxy in front yet — don't read X-Forwarded-* headers from
	// arbitrary clients. When deploying behind nginx/ALB/Cloudflare, replace
	// this with SetTrustedProxies(<known proxy CIDRs>).
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	routes.Register(router, authHandler, jobsHandler, jtiStore, db, rdb, mq)

	return &App{Router: router, DB: db, Redis: rdb, RabbitMQ: mq}, nil
}
