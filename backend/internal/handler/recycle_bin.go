package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/service"
	"github.com/nuonuo/nuonetdisk/pkg/httputil"
)

type RecycleBinHandler struct {
	recycleBinService *service.RecycleBinService
}

func NewRecycleBinHandler(recycleBinService *service.RecycleBinService) *RecycleBinHandler {
	return &RecycleBinHandler{recycleBinService: recycleBinService}
}

func (h *RecycleBinHandler) ListDeletedFiles(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}

	files, total, err := h.recycleBinService.ListDeletedFiles(c.Request.Context(), *userID, offset, limit)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, gin.H{
		"items":    files,
		"total":    total,
		"offset":   offset,
		"limit":    limit,
		"has_more": offset+len(files) < total,
	})
}

func (h *RecycleBinHandler) ListDeletedFolders(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}

	folders, total, err := h.recycleBinService.ListDeletedFolders(c.Request.Context(), *userID, offset, limit)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, gin.H{
		"items":    folders,
		"total":    total,
		"offset":   offset,
		"limit":    limit,
		"has_more": offset+len(folders) < total,
	})
}

func (h *RecycleBinHandler) RestoreFile(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid file ID")
		return
	}

	result, err := h.recycleBinService.RestoreFile(c.Request.Context(), fileID, *userID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, result)
}

func (h *RecycleBinHandler) RestoreFolder(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	folderID, err := uuid.Parse(c.Param("folderId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid folder ID")
		return
	}

	result, err := h.recycleBinService.RestoreFolder(c.Request.Context(), folderID, *userID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, result)
}

func (h *RecycleBinHandler) PermanentDeleteFile(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid file ID")
		return
	}

	if err := h.recycleBinService.PermanentDeleteFile(c.Request.Context(), fileID, *userID); err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *RecycleBinHandler) PermanentDeleteFolder(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	folderID, err := uuid.Parse(c.Param("folderId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid folder ID")
		return
	}

	if err := h.recycleBinService.PermanentDeleteFolder(c.Request.Context(), folderID, *userID); err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
