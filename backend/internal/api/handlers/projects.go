package handlers

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/etasoft/cloudcontrol/internal/container"
	"github.com/etasoft/cloudcontrol/internal/database/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProjectHandler struct {
	db     *gorm.DB
	engine *container.TemplateEngine
}

func NewProjectHandler(db *gorm.DB, engine *container.TemplateEngine) *ProjectHandler {
	return &ProjectHandler{db: db, engine: engine}
}

type createProjectRequest struct {
	Name       string            `json:"name" binding:"required"`
	Stack      string            `json:"stack" binding:"required"`
	DBName     string            `json:"db_name"`
	DBUser     string            `json:"db_user"`
	DBPassword string            `json:"db_password"`
	AppPort    string            `json:"app_port"`
	Domain     string            `json:"domain"`
	ExtraEnv   map[string]string `json:"extra_env"`
}

// List godoc — GET /api/v1/projects
func (h *ProjectHandler) List(c *gin.Context) {
	var projects []models.Project
	if err := h.db.Preload("Containers").Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects, "total": len(projects)})
}

// Get godoc — GET /api/v1/projects/:id
func (h *ProjectHandler) Get(c *gin.Context) {
	var project models.Project
	if err := h.db.Preload("Containers").Preload("SecurityLogs").First(&project, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}
	c.JSON(http.StatusOK, project)
}

// Create godoc — POST /api/v1/projects
func (h *ProjectHandler) Create(c *gin.Context) {
	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := container.ProjectParams{
		ProjectName: req.Name,
		Stack:       req.Stack,
		DBName:      req.DBName,
		DBUser:      req.DBUser,
		DBPassword:  req.DBPassword,
		AppPort:     req.AppPort,
		Domain:      req.Domain,
		ExtraEnv:    req.ExtraEnv,
	}
	if params.AppPort == "" {
		params.AppPort = "8080"
	}
	if params.DBName == "" {
		params.DBName = req.Name
	}

	files, err := h.engine.GenerateFiles(params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	compose := files["docker-compose.yml"]

	workDir := filepath.Join("projects", req.Name)
	for relPath, content := range files {
		fullPath := filepath.Join(workDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create directory for " + relPath})
			return
		}
		if err := os.WriteFile(fullPath, []byte(content), 0640); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not write " + relPath})
			return
		}
	}

	userID, _ := c.Get("user_id")
	project := models.Project{
		Name:           req.Name,
		StackType:      req.Stack,
		ComposeContent: compose,
		WorkDir:        workDir,
		Status:         models.ProjectStatusDraft,
		UserID:         userID.(string),
	}

	if err := h.db.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"project":         project,
		"compose_content": compose,
	})
}

// Up godoc — POST /api/v1/projects/:id/up
func (h *ProjectHandler) Up(c *gin.Context) {
	project, ok := h.fetchProject(c)
	if !ok {
		return
	}

	cmd := exec.CommandContext(c.Request.Context(), "docker", "compose", "-f", filepath.Join(project.WorkDir, "docker-compose.yml"), "up", "-d", "--build")
	out, err := cmd.CombinedOutput()
	if err != nil {
		h.db.Model(&project).Update("status", models.ProjectStatusError)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": string(out)})
		return
	}

	h.db.Model(&project).Update("status", models.ProjectStatusRunning)
	c.JSON(http.StatusOK, gin.H{"status": "running", "output": string(out)})
}

// Down godoc — POST /api/v1/projects/:id/down
func (h *ProjectHandler) Down(c *gin.Context) {
	project, ok := h.fetchProject(c)
	if !ok {
		return
	}

	cmd := exec.CommandContext(c.Request.Context(), "docker", "compose", "-f", filepath.Join(project.WorkDir, "docker-compose.yml"), "down")
	out, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "output": string(out)})
		return
	}

	h.db.Model(&project).Update("status", models.ProjectStatusStopped)
	c.JSON(http.StatusOK, gin.H{"status": "stopped", "output": string(out)})
}

// Delete godoc — DELETE /api/v1/projects/:id
func (h *ProjectHandler) Delete(c *gin.Context) {
	project, ok := h.fetchProject(c)
	if !ok {
		return
	}

	// Bring down first
	exec.Command("docker", "compose", "-f", filepath.Join(project.WorkDir, "docker-compose.yml"), "down", "-v").Run()

	if err := h.db.Delete(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// ListStacks godoc — GET /api/v1/stacks
func (h *ProjectHandler) ListStacks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"stacks": h.engine.ListStacks()})
}

func (h *ProjectHandler) fetchProject(c *gin.Context) (models.Project, bool) {
	var project models.Project
	if err := h.db.First(&project, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return project, false
	}
	return project, true
}
