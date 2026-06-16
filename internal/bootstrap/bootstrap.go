package bootstrap

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"jobflow/internal/auth"
	"jobflow/internal/config"
	"jobflow/internal/cron"
	"jobflow/internal/database"
	"jobflow/internal/health"
	"jobflow/internal/jobs"
	"jobflow/internal/middleware"
	"jobflow/internal/rabbitmq"
	"jobflow/internal/ratelimit"
	"jobflow/internal/redis"
	"jobflow/internal/routes"
	"jobflow/internal/sse"
)

type App struct {
	Router              *gin.Engine
	DB                  *gorm.DB
	Redis               *redis.Redis
	RabbitMQ            *rabbitmq.RabbitMQ
	TokenBucket         *ratelimit.TokenBucket
	SSESubscriber       *sse.Subscriber
	RefreshTokenCleaner *cron.RefreshTokenCleaner
	JobReconciler       *cron.JobReconciler
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

	sseManager := sse.NewClientManager()
	sseHandler := sse.NewHandler(sseManager)
	sseSubscriber := sse.NewSubscriber(rdb, sseManager)

	tokenCleaner := cron.NewRefreshTokenCleaner(db)
	jobReconciler := cron.NewJobReconciler(jobsService)
	healthHandler := health.NewHandler(db, rdb, mq)

	cfg := config.Get()
	bucket := ratelimit.NewTokenBucket(cfg.RateLimitRate, cfg.RateLimitCapacity)

	router := gin.Default()
	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	router.Use(middleware.RateLimit(bucket))
	routes.Register(router, healthHandler, authHandler, jobsHandler, sseHandler, jtiStore)

	return &App{Router: router, DB: db, Redis: rdb, RabbitMQ: mq, TokenBucket: bucket, SSESubscriber: sseSubscriber, RefreshTokenCleaner: tokenCleaner, JobReconciler: jobReconciler}, nil
}

func (a *App) Stop() {
	a.TokenBucket.Stop()
}
