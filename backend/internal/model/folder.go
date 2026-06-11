package model

import (
	"time"

	"github.com/google/uuid"
)

type Folder struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	Name           string     `json:"name"`
	ParentFolderID *uuid.UUID `json:"parent_folder_id"`
	IsDeleted      bool       `json:"is_deleted"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	Version        int        `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
