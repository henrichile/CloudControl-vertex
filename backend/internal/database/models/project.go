package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectStatus string

const (
	ProjectStatusDraft   ProjectStatus = "draft"
	ProjectStatusRunning ProjectStatus = "running"
	ProjectStatusStopped ProjectStatus = "stopped"
	ProjectStatusError   ProjectStatus = "error"
)

type Project struct {
	ID             string        `gorm:"primaryKey;type:text" json:"id"`
	Name           string        `gorm:"uniqueIndex;not null" json:"name"`
	StackType      string        `gorm:"not null" json:"stack_type"`
	ComposeContent string        `gorm:"type:text" json:"compose_content"`
	WorkDir        string        `gorm:"type:text" json:"work_dir"`
	Status         ProjectStatus `gorm:"default:draft" json:"status"`
	UserID         string        `gorm:"type:text;index" json:"user_id"`
	User           *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Containers     []Container   `gorm:"foreignKey:ProjectID" json:"containers,omitempty"`
	SecurityLogs   []SecurityLog `gorm:"foreignKey:ProjectID" json:"security_logs,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}
