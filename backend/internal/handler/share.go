package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/internal/service"
	"github.com/nuonuo/nuonetdisk/pkg/httputil"
)

type ShareHandler struct {
	shareService  *service.ShareService
	fileService   *service.FileService
	folderService *service.FolderService
}

func NewShareHandler(shareService *service.ShareService, fileService *service.FileService, folderService *service.FolderService) *ShareHandler {
	return &ShareHandler{
		shareService:  shareService,
		fileService:   fileService,
		folderService: folderService,
	}
}

type createShareRequest struct {
	ResourceType string     `json:"resource_type" binding:"required"`
	ResourceID   string     `json:"resource_id" binding:"required"`
	Permission   string     `json:"permission" binding:"required"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

func (h *ShareHandler) Create(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	var req createShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid request")
		return
	}

	resourceID, err := uuid.Parse(req.ResourceID)
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid resource_id")
		return
	}

	var resourceType model.ResourceType
	switch req.ResourceType {
	case "file":
		resourceType = model.ResourceTypeFile
	case "folder":
		resourceType = model.ResourceTypeFolder
	default:
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "resource_type must be 'file' or 'folder'")
		return
	}

	var permission model.SharePermission
	switch req.Permission {
	case "read":
		permission = model.SharePermissionRead
	case "write":
		permission = model.SharePermissionWrite
	default:
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "permission must be 'read' or 'write'")
		return
	}

	result, err := h.shareService.CreateShareLink(c.Request.Context(), service.CreateShareInput{
		UserID:       *userID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Permission:   permission,
		ExpiresAt:    req.ExpiresAt,
	})
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusCreated, result)
}

func (h *ShareHandler) Revoke(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	shareID, err := uuid.Parse(c.Param("shareId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid share ID")
		return
	}

	if err := h.shareService.Revoke(c.Request.Context(), shareID, *userID); err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ShareHandler) AccessByToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		httputil.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Share link not found")
		return
	}

	link, err := h.shareService.GetByToken(c.Request.Context(), token)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	if link.ResourceType == model.ResourceTypeFile {
		file, err := h.fileService.GetFile(c.Request.Context(), link.ResourceID, link.UserID)
		if err != nil {
			httputil.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Shared resource not found")
			return
		}
		httputil.RespondJSON(c, http.StatusOK, file)
	} else {
		folders, err := h.folderService.ListByParent(c.Request.Context(), link.UserID, &link.ResourceID)
		if err != nil {
			httputil.RespondError(c, http.StatusNotFound, "NOT_FOUND", "Shared resource not found")
			return
		}
		httputil.RespondJSON(c, http.StatusOK, gin.H{
			"folder_id": link.ResourceID,
			"contents":  folders,
		})
	}
}
