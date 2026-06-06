// File: internal/sse/handler.go
package sse

import (
	"io"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	manager *ClientManager
}

func NewHandler(manager *ClientManager) *Handler {
	return &Handler{manager: manager}
}

// SSEStream registers the caller as an SSE client and streams events until
// the request context is cancelled (client disconnects or server shuts down).
func (h *Handler) SSEStream(c *gin.Context) {
	userId := c.Param("userId")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	clientCh := h.manager.AddClient(userId)
	defer h.manager.RemoveClient(userId)

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-clientCh:
			if !ok {
				return false
			}
			c.SSEvent("job", msg)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
