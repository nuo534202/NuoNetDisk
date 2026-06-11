package model

import (
	"time"

	"github.com/google/uuid"
)

type File struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	Name           string     `json:"name"`
	ObjectKey      string     `json:"-"`
	Size           int64      `json:"size"`
	MimeType       string     `json:"mime_type"`
	SHA256Hash     string     `json:"sha256_hash"`
	ParentFolderID *uuid.UUID `json:"parent_folder_id"`
	IsDeleted      bool       `json:"is_deleted"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	Version        int        `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
