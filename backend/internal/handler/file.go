package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/service"
	"github.com/nuonuo/nuonetdisk/pkg/httputil"
)

type FileHandler struct {
	fileService *service.FileService
}

func NewFileHandler(fileService *service.FileService) *FileHandler {
	return &FileHandler{fileService: fileService}
}

func (h *FileHandler) List(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	var folderID *uuid.UUID
	if fidStr := c.Query("folder_id"); fidStr != "" {
		fid, err := uuid.Parse(fidStr)
		if err != nil {
			httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid folder_id")
			return
		}
		folderID = &fid
	}

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit > 100 {
		limit = 100
	}
	sort := c.DefaultQuery("sort", "name")
	order := c.DefaultQuery("order", "asc")

	files, total, err := h.fileService.ListFiles(c.Request.Context(), *userID, folderID, offset, limit, sort, order)
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

func (h *FileHandler) Upload(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	var parentFolderID *uuid.UUID
	if pfIDStr := c.PostForm("parent_folder_id"); pfIDStr != "" {
		pfID, err := uuid.Parse(pfIDStr)
		if err != nil {
			httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid parent_folder_id")
			return
		}
		parentFolderID = &pfID
	}

	file, err := c.FormFile("file")
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "File is required")
		return
	}

	result, err := h.fileService.Upload(c.Request.Context(), *userID, file, parentFolderID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusCreated, result)
}

func (h *FileHandler) GetByID(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid file ID")
		return
	}

	result, err := h.fileService.GetFile(c.Request.Context(), fileID, *userID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, result)
}

func (h *FileHandler) Download(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid file ID")
		return
	}

	reader, fileName, fileSize, err := h.fileService.Download(c.Request.Context(), fileID, *userID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Header("Content-Length", strconv.FormatInt(fileSize, 10))
	c.DataFromReader(http.StatusOK, fileSize, "application/octet-stream", reader, nil)
}

func (h *FileHandler) Update(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid file ID")
		return
	}

	var req struct {
		Name           *string    `json:"name"`
		ParentFolderID *uuid.UUID `json:"parent_folder_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid request body")
		return
	}

	result, err := h.fileService.Update(c.Request.Context(), fileID, *userID, req.Name, req.ParentFolderID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, result)
}

func (h *FileHandler) Delete(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	fileID, err := uuid.Parse(c.Param("fileId"))
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid file ID")
		return
	}

	if err := h.fileService.Delete(c.Request.Context(), fileID, *userID); err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func getUserID(c *gin.Context) *uuid.UUID {
	userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		httputil.RespondError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required")
		return nil
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Invalid user")
		return nil
	}

	return &userID
}
