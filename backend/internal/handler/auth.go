package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/service"
	"github.com/nuonuo/nuonetdisk/pkg/httputil"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid email or password")
		return
	}

	user, err := h.authService.Register(c.Request.Context(), service.AuthRegisterInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusCreated, gin.H{
		"id":           user.ID,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"created_at":   user.CreatedAt.Format(time.RFC3339),
		"updated_at":   user.UpdatedAt.Format(time.RFC3339),
	})
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid email or password")
		return
	}

	pair, _, err := h.authService.Login(c.Request.Context(), service.AuthLoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, pair)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid refresh token")
		return
	}

	pair, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, pair)
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid refresh token")
		return
	}

	userID := c.GetString("user_id")
	uid, err := uuid.Parse(userID)
	if err != nil {
		httputil.RespondError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Invalid user")
		return
	}

	if err := h.authService.Logout(c.Request.Context(), uid, req.RefreshToken); err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
