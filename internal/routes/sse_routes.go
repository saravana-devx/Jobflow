package routes

import (
	"github.com/gin-gonic/gin"

	"jobflow/internal/auth"
	"jobflow/internal/middleware"
	"jobflow/internal/sse"
)

func RegisterSSERoutes(r *gin.Engine, h *sse.Handler, jtiStore *auth.JTIStore) {
	r.GET("/subscribe/:userId", middleware.RequireAuth(jtiStore), h.SSEStream)
}
