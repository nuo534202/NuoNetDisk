package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/internal/repository"
	"github.com/nuonuo/nuonetdisk/internal/service"
	"github.com/nuonuo/nuonetdisk/pkg/httputil"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func (h *AdminHandler) GetDashboard(c *gin.Context) {
	dashboard, err := h.adminService.GetDashboard(c.Request.Context())
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}
	httputil.RespondJSON(c, http.StatusOK, dashboard)
}

type userListQuery struct {
	Offset       int    `form:"offset"`
	Limit        int    `form:"limit"`
	Q            string `form:"q"`
	EmailFilter  string `form:"email_filter"`
	NameFilter   string `form:"name_filter"`
	RoleFilter   string `form:"role_filter"`
	GenderFilter string `form:"gender_filter"`
	DateFrom     string `form:"date_from"`
	DateTo       string `form:"date_to"`
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	var q userListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid query parameters")
		return
	}
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 20
	}

	filter := repository.UserFilter{
		Q:            q.Q,
		EmailFilter:  q.EmailFilter,
		NameFilter:   q.NameFilter,
		RoleFilter:   q.RoleFilter,
		GenderFilter: q.GenderFilter,
		DateFrom:     q.DateFrom,
		DateTo:       q.DateTo,
	}
	users, total, err := h.adminService.ListUsers(c.Request.Context(), q.Offset, q.Limit, filter)
	if err != nil {
		httputil.RespondServiceError(c, err)
		return
	}

	httputil.RespondJSON(c, http.StatusOK, gin.H{
		"items":    users,
		"total":    total,
		"offset":   q.Offset,
		"limit":    q.Limit,
		"has_more": q.Offset+q.Limit < total,
	})
}

type registerAdminRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AdminHandler) RegisterAdmin(c *gin.Context) {
	var req registerAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Email and password are required")
		return
	}
	if len(req.Password) < 8 {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Password must be at least 8 characters")
		return
	}
	if err := h.adminService.RegisterAdmin(c.Request.Context(), req.Email, req.Password); err != nil {
		httputil.RespondServiceError(c, err)
		return
	}
	httputil.RespondJSON(c, http.StatusCreated, gin.H{})
}

// GetUserAvatar streams a specific user's avatar. Admin-only.
func (h *AdminHandler) GetUserAvatar(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		httputil.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "Invalid user ID")
		return
	}

	reader, contentType, size, err := h.adminService.GetUserAvatar(c.Request.Context(), userID)
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
