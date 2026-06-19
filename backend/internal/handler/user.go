package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nuonuo/nuonetdisk/internal/model"

	"github.com/nuonuo/nuonetdisk/internal/service"
	"github.com/nuonuo/nuonetdisk/pkg/httputil"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type UpdateMeRequest struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Bio         *string `json:"bio"`
	Gender      *string `json:"gender"`
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	user, err := h.userService.GetMe(c.Request.Context(), *userID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, 200, user)
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid request body")
		return
	}

	if req.DisplayName != nil {
		trimmed := *req.DisplayName
		if len(trimmed) == 0 || len(trimmed) > 100 {
			httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Display name must be between 1 and 100 characters")
			return
		}
	}
	if req.Bio != nil && len(*req.Bio) > 2000 {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Bio must be at most 2000 characters")
		return
	}
	if req.Gender != nil {
		gender := *req.Gender
		if gender != "" && gender != "male" && gender != "female" {
			httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Gender must be one of: male, female")
			return
		}
	}

	user, err := h.userService.UpdateMe(c.Request.Context(), *userID, service.UpdateMeInput{
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
		Bio:         req.Bio,
		Gender:      req.Gender,
	})
	if err != nil {
		if err == model.ErrNotFound {
			httputil.RespondError(c, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		httputil.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update profile")
		return
	}

	httputil.RespondJSON(c, http.StatusOK, user)
}

func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Avatar file is required")
		return
	}

	user, err := h.userService.UpdateAvatar(c.Request.Context(), *userID, file)
	if err != nil {
		if err == model.ErrFileTooLarge {
			httputil.RespondError(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Avatar must be less than 5MB")
			return
		}
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, user)
}

func (h *UserHandler) DownloadAvatar(c *gin.Context) {
	userID := getUserID(c)
	if userID == nil {
		return
	}

	reader, contentType, size, err := h.userService.GetAvatar(c.Request.Context(), *userID)
	if err != nil {
		if err == model.ErrNotFound {
			httputil.RespondError(c, http.StatusNotFound, "NOT_FOUND", "No avatar set")
			return
		}
		httputil.RespondServiceError(c, err)
		return
	}
	defer reader.Close()

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(size, 10))
	c.Header("Cache-Control", "private, max-age=3600")
	c.DataFromReader(http.StatusOK, size, contentType, reader, nil)
}
