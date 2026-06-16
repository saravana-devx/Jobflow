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

func (h *Handler) SSEStream(c *gin.Context) {
	userId := c.Param("userId")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	connID, clientCh := h.manager.AddClient(userId)
	defer h.manager.RemoveClient(userId, connID)

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
