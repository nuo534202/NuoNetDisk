package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/service"
	httputil "github.com/nuonuo/nuonetdisk/pkg/httputil"
	"github.com/nuonuo/nuonetdisk/pkg/nullable"
)

type FolderHandler struct {
	folderService *service.FolderService
}

func NewFolderHandler(folderService *service.FolderService) *FolderHandler {
	return &FolderHandler{folderService: folderService}
}

type createFolderRequest struct {
	Name           string     `json:"name" binding:"required"`
	ParentFolderID *uuid.UUID `json:"parent_folder_id"`
}

func (h *FolderHandler) Create(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	var req createFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Name is required")
		return
	}

	result, err := h.folderService.Create(c.Request.Context(), *userID, req.Name, req.ParentFolderID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusCreated, result)
}

func (h *FolderHandler) List(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	var parentID *uuid.UUID
	if pIDStr := c.Query("parent_id"); pIDStr != "" {
		pID, err := uuid.Parse(pIDStr)
		if err != nil {
			httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid parent_id")
			return
		}
		parentID = &pID
	}

	folders, err := h.folderService.ListByParent(c.Request.Context(), *userID, parentID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, gin.H{
		"items": folders,
		"total": len(folders),
	})
}

func (h *FolderHandler) GetByID(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	folderID, err := uuid.Parse(c.Param("folderId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid folder ID")
		return
	}

	result, err := h.folderService.GetByID(c.Request.Context(), folderID, *userID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, result)
}

func (h *FolderHandler) GetAncestors(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	folderID, err := uuid.Parse(c.Param("folderId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid folder ID")
		return
	}

	result, err := h.folderService.GetAncestors(c.Request.Context(), folderID, *userID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, result)
}

type updateFolderRequest struct {
	Name           *string       `json:"name"`
	ParentFolderID nullable.UUID `json:"parent_folder_id"`
}

func (h *FolderHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	folderID, err := uuid.Parse(c.Param("folderId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid folder ID")
		return
	}

	var req updateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid request body")
		return
	}

	result, err := h.folderService.Update(c.Request.Context(), folderID, *userID, req.Name, req.ParentFolderID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, result)
}

func (h *FolderHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	folderID, err := uuid.Parse(c.Param("folderId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid folder ID")
		return
	}

	if err := h.folderService.Delete(c.Request.Context(), folderID, *userID); err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
