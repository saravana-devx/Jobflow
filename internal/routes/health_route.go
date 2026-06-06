package routes

import (
	"github.com/gin-gonic/gin"

	"jobflow/internal/health"
)

func RegisterHealthRoute(r *gin.Engine, h *health.Handler) {
	r.GET("/health", h.Health)
}
