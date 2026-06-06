package routes

import (
	"github.com/gin-gonic/gin"
	"jobflow/internal/auth"
	"jobflow/internal/middleware"
)

func RegisterAuthRoutes(r *gin.Engine, h *auth.Handler, jtiStore *auth.JTIStore) {
	g := r.Group("/auth")
	{
		g.POST("/sign-up", h.CreateUser)
		g.POST("/login", h.LoginUser)
		g.POST("/refresh", h.RefreshAccessToken)
		g.POST("/logout", middleware.RequireAuth(jtiStore), h.Logout)
	}
}
