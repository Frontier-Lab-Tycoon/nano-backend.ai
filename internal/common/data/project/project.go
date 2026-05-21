// Package project defines the Project entity that wraps related runs.
package project

import (
	"time"

	"github.com/google/uuid"
)

// Project groups related run specs and executions.
type Project struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// New creates a Project with a generated ID and creation timestamp.
func New(name, description string) Project {
	return Project{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}
}
