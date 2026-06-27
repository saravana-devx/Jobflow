package cron

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"jobflow/internal/auth"
	redisx "jobflow/internal/redis"
)

const cleanupInterval = 1 * time.Hour

// hold the lease for most of the interval so only one replica runs the cleanup
const cleanupLeaseTTL = 55 * time.Minute

type RefreshTokenCleaner struct {
	db  *gorm.DB
	rdb *redisx.Redis
}

func NewRefreshTokenCleaner(db *gorm.DB, rdb *redisx.Redis) *RefreshTokenCleaner {
	return &RefreshTokenCleaner{db: db, rdb: rdb}
}

func (c *RefreshTokenCleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	log.Println("RefreshTokenCleaner started")
	for {
		select {
		case <-ticker.C:
			runIfLeader(ctx, c.rdb, "cron:refresh-token-cleaner", cleanupLeaseTTL, func() {
				c.clean(ctx)
			})
		case <-ctx.Done():
			log.Println("RefreshTokenCleaner stopped")
			return
		}
	}
}

func (c *RefreshTokenCleaner) clean(ctx context.Context) {
	result := c.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&auth.RefreshToken{})
	if result.Error != nil {
		log.Printf("RefreshTokenCleaner: cleanup failed: %v", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		log.Printf("RefreshTokenCleaner: deleted %d expired tokens", result.RowsAffected)
	}
}
