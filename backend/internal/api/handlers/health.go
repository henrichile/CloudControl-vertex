package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	dockerPing func(ctx context.Context) error
	ollamaPing func(ctx context.Context) error
}

func NewHealthHandler(dockerPing, ollamaPing func(ctx context.Context) error) *HealthHandler {
	return &HealthHandler{dockerPing: dockerPing, ollamaPing: ollamaPing}
}

func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	dockerOK := h.dockerPing(ctx) == nil
	ollamaOK := h.ollamaPing(ctx) == nil

	status := "ok"
	code := http.StatusOK
	if !dockerOK {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, gin.H{
		"status":    status,
		"timestamp": time.Now().UTC(),
		"services": gin.H{
			"docker": boolToStatus(dockerOK),
			"ollama": boolToStatus(ollamaOK),
		},
	})
}

func boolToStatus(ok bool) string {
	if ok {
		return "up"
	}
	return "down"
}
