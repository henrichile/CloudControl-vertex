package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

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

	projectsDir := os.Getenv("PROJECTS_DIR")
	if projectsDir == "" {
		projectsDir = "/opt/cloudcontrol/projects"
	}
	workDir := filepath.Join(projectsDir, req.Name)
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

// UpStream godoc — GET /api/v1/projects/:id/up/stream (Server-Sent Events)
// Ejecuta docker compose up --build y emite cada línea de output como evento SSE.
func (h *ProjectHandler) UpStream(c *gin.Context) {
	project, ok := h.fetchProject(c)
	if !ok {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // deshabilitar buffer de nginx/traefik
	c.Writer.WriteHeader(http.StatusOK)

	flusher, canFlush := c.Writer.(http.Flusher)

	sendEvent := func(typ, msg string) {
		b, _ := json.Marshal(map[string]string{"type": typ, "msg": msg})
		fmt.Fprintf(c.Writer, "data: %s\n\n", b)
		if canFlush {
			flusher.Flush()
		}
	}

	sendEvent("log", "▶  docker compose up --build")

	cmd := exec.CommandContext(c.Request.Context(),
		"docker", "compose",
		"-f", filepath.Join(project.WorkDir, "docker-compose.yml"),
		"up", "-d", "--build",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sendEvent("error", "stdout pipe: "+err.Error())
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		sendEvent("error", "stderr pipe: "+err.Error())
		return
	}

	if err := cmd.Start(); err != nil {
		sendEvent("error", err.Error())
		return
	}

	// Leer stdout y stderr en goroutines separadas; enviar líneas a canal para
	// escribir en el response desde el goroutine principal (evita race en Writer).
	logCh := make(chan string, 256)
	var wg sync.WaitGroup

	scanPipe := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 1024*64), 1024*64)
		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), "\r")
			if line != "" {
				logCh <- line
			}
		}
	}

	wg.Add(2)
	go scanPipe(stdout)
	go scanPipe(stderr)
	go func() { wg.Wait(); close(logCh) }()

	for line := range logCh {
		sendEvent("log", line)
	}

	if err := cmd.Wait(); err != nil {
		h.db.Model(&project).Update("status", models.ProjectStatusError)
		sendEvent("error", err.Error())
		return
	}

	h.db.Model(&project).Update("status", models.ProjectStatusRunning)
	sendEvent("done", "running")
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
