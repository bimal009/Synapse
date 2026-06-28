package models

import (
	"time"

	"github.com/bimal009/Synapse/configs"
)

type Project struct {
	ID         string     `json:"id" db:"id"`
	Name       string     `json:"name" db:"name"`
	Path       string     `json:"path" db:"path"`
	Trusted    bool       `json:"trusted" db:"trusted"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	LastOpened *time.Time `json:"last_opened,omitempty" db:"last_opened"`
}

type ProjectConfig struct {
	ProjectID string                         `json:"project_id" db:"project_id"`
	Models    map[string]configs.ModelConfig `json:"models" db:"models"`
}
