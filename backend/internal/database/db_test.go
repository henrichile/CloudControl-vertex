package database_test

import (
	"testing"

	"github.com/etasoft/cloudcontrol/internal/database"
	"github.com/etasoft/cloudcontrol/internal/database/models"
)

func TestConnect_InMemory(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("DB ping error: %v", err)
	}
}

func TestAutoMigrate_TablesExist(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	tables := []string{"users", "projects", "containers", "security_logs"}
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			t.Errorf("expected table %q to exist after migration", table)
		}
	}
}

func TestUserCRUD(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	user := models.User{
		Email:        "test@example.com",
		PasswordHash: "hashed",
		Role:         models.RoleAdmin,
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("Create user error: %v", err)
	}
	if user.ID == "" {
		t.Error("user ID should be auto-generated")
	}

	var found models.User
	if err := db.First(&found, "email = ?", "test@example.com").Error; err != nil {
		t.Fatalf("Find user error: %v", err)
	}
	if found.Email != user.Email {
		t.Errorf("expected email %q, got %q", user.Email, found.Email)
	}
	if found.Role != models.RoleAdmin {
		t.Errorf("expected role admin, got %q", found.Role)
	}
}

func TestProjectCRUD(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	user := models.User{Email: "owner@test.com", PasswordHash: "x", Role: models.RoleOperator}
	db.Create(&user)

	project := models.Project{
		Name:           "mi-proyecto",
		StackType:      "MERN",
		ComposeContent: "services: {}",
		Status:         models.ProjectStatusDraft,
		UserID:         user.ID,
	}
	if err := db.Create(&project).Error; err != nil {
		t.Fatalf("Create project error: %v", err)
	}

	db.Model(&project).Update("status", models.ProjectStatusRunning)

	var found models.Project
	db.First(&found, "id = ?", project.ID)
	if found.Status != models.ProjectStatusRunning {
		t.Errorf("expected status 'running', got %q", found.Status)
	}
}

func TestSecurityLogCRUD(t *testing.T) {
	db, err := database.Connect(":memory:")
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}

	log := models.SecurityLog{
		ProjectID:  "proj-123",
		Severity:   models.SeverityCritical,
		Finding:    "Contraseña expuesta en .env",
		Suggestion: "Usar Docker secrets",
		FilePath:   ".env",
		LineNumber: 3,
	}
	if err := db.Create(&log).Error; err != nil {
		t.Fatalf("Create security log error: %v", err)
	}

	var found models.SecurityLog
	db.First(&found, "project_id = ?", "proj-123")
	if found.Severity != models.SeverityCritical {
		t.Errorf("expected severity 'critical', got %q", found.Severity)
	}
	if found.Resolved {
		t.Error("log should not be resolved by default")
	}
}
