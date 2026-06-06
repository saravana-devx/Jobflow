package routes

import (
	"github.com/gin-gonic/gin"
	"jobflow/internal/auth"
	"jobflow/internal/jobs"
	"jobflow/internal/middleware"
)

func RegisterJobsRoutes(r *gin.Engine, h *jobs.Handler, jtiStore *auth.JTIStore) {
	g := r.Group("/jobs", middleware.RequireAuth(jtiStore))
	{
		// g.POST("", h.CreateJob)
		// g.POST("/bulk", h.CreateJobs)
		g.POST("/", h.CreateJobs)
		g.GET("/:id", h.GetJobById)
		g.GET("", h.GetAllJobs)
		g.PATCH("/:id", h.UpdateJob)
		g.DELETE("/:id", h.DeleteJob)
	}
}
