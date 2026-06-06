package cron

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"jobflow/internal/auth"
)

const cleanupInterval = 1 * time.Hour

type RefreshTokenCleaner struct {
	db *gorm.DB
}

func NewRefreshTokenCleaner(db *gorm.DB) *RefreshTokenCleaner {
	return &RefreshTokenCleaner{db: db}
}

func (c *RefreshTokenCleaner) Start(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	log.Println("RefreshTokenCleaner started")
	for {
		select {
		case <-ticker.C:
			c.clean(ctx)
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
