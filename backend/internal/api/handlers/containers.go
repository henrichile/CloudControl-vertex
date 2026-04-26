package handlers

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/etasoft/cloudcontrol/internal/container"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type ContainerHandler struct {
	mgr *container.Manager
}

func NewContainerHandler(mgr *container.Manager) *ContainerHandler {
	return &ContainerHandler{mgr: mgr}
}

// List godoc — GET /api/v1/containers
func (h *ContainerHandler) List(c *gin.Context) {
	onlyRunning := c.Query("running") == "true"
	containers, err := h.mgr.List(c.Request.Context(), onlyRunning)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"containers": containers, "total": len(containers)})
}

// Inspect godoc — GET /api/v1/containers/:id
func (h *ContainerHandler) Inspect(c *gin.Context) {
	info, err := h.mgr.Inspect(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// Start godoc — POST /api/v1/containers/:id/start
func (h *ContainerHandler) Start(c *gin.Context) {
	if err := h.mgr.Start(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

// Stop godoc — POST /api/v1/containers/:id/stop
func (h *ContainerHandler) Stop(c *gin.Context) {
	var opts struct {
		Timeout int `json:"timeout"`
	}
	_ = c.ShouldBindJSON(&opts)

	var timeout *int
	if opts.Timeout > 0 {
		timeout = &opts.Timeout
	}

	if err := h.mgr.Stop(c.Request.Context(), c.Param("id"), timeout); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
}

// Remove godoc — DELETE /api/v1/containers/:id
func (h *ContainerHandler) Remove(c *gin.Context) {
	force := c.Query("force") == "true"
	if err := h.mgr.Remove(c.Request.Context(), c.Param("id"), force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "removed"})
}

// Logs godoc — GET /api/v1/containers/:id/logs
func (h *ContainerHandler) Logs(c *gin.Context) {
	tail := c.DefaultQuery("tail", "100")
	follow := c.Query("follow") == "true"

	rc, err := h.mgr.Logs(c.Request.Context(), c.Param("id"), tail, follow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rc.Close()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	io.Copy(c.Writer, rc)
}

// Stats godoc — GET /api/v1/containers/:id/stats (snapshot)
func (h *ContainerHandler) Stats(c *gin.Context) {
	info, err := h.mgr.Stats(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// StatsWS godoc — GET /api/v1/containers/:id/stats/ws (WebSocket stream)
func (h *ContainerHandler) StatsWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	intervalSec := 2
	if v := c.Query("interval"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			intervalSec = n
		}
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// Cancel on client disconnect
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cancel()
				return
			}
		}
	}()

	ch := make(chan container.ContainerInfo, 4)
	go h.mgr.StreamStats(ctx, c.Param("id"), ch, time.Duration(intervalSec)*time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case info := <-ch:
			if err := conn.WriteJSON(info); err != nil {
				return
			}
		}
	}
}

// UpdateLimits godoc — PATCH /api/v1/containers/:id/limits
func (h *ContainerHandler) UpdateLimits(c *gin.Context) {
	var limits container.ResourceLimits
	if err := c.ShouldBindJSON(&limits); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.mgr.UpdateLimits(c.Request.Context(), c.Param("id"), limits); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "limits updated", "limits": limits})
}
