package routes

import (
	"github.com/gin-gonic/gin"

	"jobflow/internal/auth"
	"jobflow/internal/health"
	"jobflow/internal/jobs"
	"jobflow/internal/sse"
)

func Register(r *gin.Engine, healthHandler *health.Handler, authHandler *auth.Handler, jobsHandler *jobs.Handler, sseHandler *sse.Handler, jtiStore *auth.JTIStore) {
	RegisterHealthRoute(r, healthHandler)
	RegisterAuthRoutes(r, authHandler, jtiStore)
	RegisterJobsRoutes(r, jobsHandler, jtiStore)
	RegisterSSERoutes(r, sseHandler, jtiStore)
}
