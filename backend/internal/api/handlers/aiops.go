package handlers

import (
	"net/http"

	"github.com/etasoft/cloudcontrol/internal/aiops"
	"github.com/etasoft/cloudcontrol/internal/container"
	"github.com/etasoft/cloudcontrol/internal/database/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AIOpsHandler struct {
	analyzer *aiops.Analyzer
	mgr      *container.Manager
	db       *gorm.DB
}

func NewAIOpsHandler(analyzer *aiops.Analyzer, mgr *container.Manager, db *gorm.DB) *AIOpsHandler {
	return &AIOpsHandler{analyzer: analyzer, mgr: mgr, db: db}
}

// AnalyzeContainer godoc — POST /api/v1/aiops/analyze
func (h *AIOpsHandler) AnalyzeContainer(c *gin.Context) {
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.mgr.Stats(c.Request.Context(), req.ContainerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch stats: " + err.Error()})
		return
	}

	snap := aiops.MetricsSnapshot{
		ContainerName: req.ContainerID,
		CPUPercent:    stats.CPUPercent,
		MemUsageMB:    stats.MemUsageMB,
		MemLimitMB:    stats.MemLimitMB,
		NetRxMB:       stats.NetRxMB,
		NetTxMB:       stats.NetTxMB,
	}

	rec, err := h.analyzer.AnalyzeMetrics(c.Request.Context(), snap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Auto-apply scaling if action is scale_up/scale_down and limits are provided
	if rec.Action != "ok" && (rec.NewCPULimit > 0 || rec.NewMemLimMB > 0) {
		limits := container.ResourceLimits{
			MemoryMB: int64(rec.NewMemLimMB),
		}
		if rec.NewCPULimit > 0 {
			// Convert CPU percent to quota (100ms period = 100000 microseconds)
			limits.CPUQuota = int64(rec.NewCPULimit * 1000)
		}
		if err := h.mgr.UpdateLimits(c.Request.Context(), req.ContainerID, limits); err != nil {
			rec.Reason += " (WARNING: auto-apply failed: " + err.Error() + ")"
		} else {
			rec.Reason += " (limits auto-applied)"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"container":      req.ContainerID,
		"metrics":        snap,
		"recommendation": rec,
	})
}

// AuditConfig godoc — POST /api/v1/aiops/audit
func (h *AIOpsHandler) AuditConfig(c *gin.Context) {
	var req struct {
		ProjectID string `json:"project_id"`
		FileName  string `json:"file_name" binding:"required"`
		Content   string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.analyzer.AuditConfig(c.Request.Context(), req.FileName, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Persist findings to database
	if req.ProjectID != "" {
		for _, f := range result.Findings {
			log := models.SecurityLog{
				ProjectID:  req.ProjectID,
				Severity:   models.Severity(f.Severity),
				Finding:    f.Finding,
				Suggestion: f.Suggestion,
				AIAnalysis: result.RawAnalysis,
				FilePath:   req.FileName,
				LineNumber: f.LineNumber,
			}
			h.db.Create(&log)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"file":    req.FileName,
		"result":  result,
	})
}

// AnalyzeLogs godoc — POST /api/v1/aiops/logs
func (h *AIOpsHandler) AnalyzeLogs(c *gin.Context) {
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
		Tail        string `json:"tail"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Tail == "" {
		req.Tail = "200"
	}

	rc, err := h.mgr.Logs(c.Request.Context(), req.ContainerID, req.Tail, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch logs: " + err.Error()})
		return
	}
	defer rc.Close()

	buf := make([]byte, 32*1024)
	n, _ := rc.Read(buf)
	logs := string(buf[:n])

	analysis, err := h.analyzer.AnalyzeLogs(c.Request.Context(), req.ContainerID, logs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"container": req.ContainerID,
		"analysis":  analysis,
	})
}
