package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Container struct {
	ID          string    `gorm:"primaryKey;type:text" json:"id"`
	ProjectID   string    `gorm:"type:text;index" json:"project_id"`
	DockerID    string    `gorm:"type:text;index" json:"docker_id"`
	Name        string    `gorm:"not null" json:"name"`
	Image       string    `json:"image"`
	Status      string    `json:"status"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemUsageMB  float64   `json:"mem_usage_mb"`
	MemLimitMB  float64   `json:"mem_limit_mb"`
	NetRxMB     float64   `json:"net_rx_mb"`
	NetTxMB     float64   `json:"net_tx_mb"`
	CPULimit    float64   `json:"cpu_limit"`
	MemLimitSet float64   `json:"mem_limit_set"`
	Ports       string    `gorm:"type:text" json:"ports"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}

func (c *Container) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}
