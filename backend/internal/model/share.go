package model

import (
	"time"

	"github.com/google/uuid"
)

type ResourceType string

const (
	ResourceTypeFile   ResourceType = "file"
	ResourceTypeFolder ResourceType = "folder"
)

type SharePermission string

const (
	SharePermissionRead  SharePermission = "read"
	SharePermissionWrite SharePermission = "write"
)

type ShareLink struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	ResourceType ResourceType    `json:"resource_type"`
	ResourceID   uuid.UUID       `json:"resource_id"`
	Token        string          `json:"token"`
	Permission   SharePermission `json:"permission"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
	IsRevoked    bool            `json:"is_revoked"`
	CreatedAt    time.Time       `json:"created_at"`
}
