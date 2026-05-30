package redis

import (
	"sync"

	"github.com/redis/go-redis/v9"

	"pulseDashboard/internal/config"
)

type Redis struct {
	Client *redis.Client
}

var (
	redisInstance *Redis
	once          sync.Once
)

// singleton pattern guarantees that this redis connection will be created only once
// every goroutine / function will get the same pointer reference
// without singleton pattern we might get multiple redis connections
func NewRedis() *Redis {
	once.Do(func() {
		c := config.Get()
		rdb := redis.NewClient(&redis.Options{
			Addr:     c.RedisAddr,
			Password: c.RedisPassword,
			DB:       0,
		})
		redisInstance = &Redis{Client: rdb}
	})
	return redisInstance
}

