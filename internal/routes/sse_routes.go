// File: internal/routes/sse_routes.go
package routes

import (
	"github.com/gin-gonic/gin"

	"jobflow/internal/sse"
)

func RegisterSSERoutes(r *gin.Engine, h *sse.Handler) {
	r.GET("/subscribe/:userId", h.SSEStream)
}
