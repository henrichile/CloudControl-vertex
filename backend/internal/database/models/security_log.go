package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

type SecurityLog struct {
	ID         string    `gorm:"primaryKey;type:text" json:"id"`
	ProjectID  string    `gorm:"type:text;index" json:"project_id"`
	Severity   Severity  `gorm:"not null" json:"severity"`
	Finding    string    `gorm:"type:text;not null" json:"finding"`
	Suggestion string    `gorm:"type:text" json:"suggestion"`
	AIAnalysis string    `gorm:"type:text" json:"ai_analysis"`
	FilePath   string    `json:"file_path"`
	LineNumber int       `json:"line_number"`
	Resolved   bool      `gorm:"default:false" json:"resolved"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *SecurityLog) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}
