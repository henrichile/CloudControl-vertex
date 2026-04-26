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

	// El backend siempre responde 200 si puede atender requests.
	// Docker y Ollama son dependencias opcionales que se reportan como
	// información pero no determinan la salud del proceso en sí.
	// Usar 503 aquí causa un deadlock: Docker Compose no marca el
	// contenedor como "healthy" hasta recibir 200, pero el contenedor
	// necesita estar "healthy" para que el frontend levante.
	status := "ok"
	if !dockerOK || !ollamaOK {
		status = "degraded"
	}

	c.JSON(http.StatusOK, gin.H{
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
