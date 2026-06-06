package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"jobflow/internal/auth"
	"jobflow/internal/httpx"
)

func RequireAuth(jtiStore *auth.JTIStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := auth.ExtractBearerToken(c.GetHeader("Authorization"))
		if err != nil {
			httpx.AbortError(c, http.StatusUnauthorized, err.Error())
			return
		}

		claims, err := auth.ParseAccessToken(token)
		if err != nil {
			httpx.AbortError(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		revoked, err := jtiStore.IsRevoked(ctx, claims.JTI)
		if err != nil {
			httpx.AbortError(c, http.StatusInternalServerError, "auth check failed")
			return
		}
		if revoked {
			httpx.AbortError(c, http.StatusUnauthorized, "token revoked")
			return
		}

		auth.SetUserID(c, claims.UserID)
		auth.SetJTI(c, claims.JTI)
		auth.SetExp(c, claims.ExpiresAt)
		c.Next()
	}
}
