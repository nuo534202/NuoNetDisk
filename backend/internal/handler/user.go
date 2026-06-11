package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	DisplayName string `json:"display_name" binding:"required"`
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondError(c, 401, "UNAUTHENTICATED", "Invalid user")
		return
	}

	user, err := h.userService.GetMe(c.Request.Context(), userID)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, 200, user)
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Invalid user")
		return
	}

	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Display name is required")
		return
	}
	if len(req.DisplayName) == 0 || len(req.DisplayName) > 100 {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Display name must be between 1 and 100 characters")
		return
	}

	user, err := h.userService.UpdateMe(c.Request.Context(), userID, req.DisplayName)
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
