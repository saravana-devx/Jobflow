package middleware

import (
	"net/http"

	"jobflow/internal/ratelimit"

	"github.com/gin-gonic/gin"
)

func RateLimit(tb *ratelimit.TokenBucket) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !tb.TryAcquire() {
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}
