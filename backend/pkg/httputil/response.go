package httputil

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nuonuo/nuonetdisk/internal/model"
)

type ErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func RespondJSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}

func RespondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	})
}

func RespondServiceError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, model.ErrNotFound):
		RespondError(c, http.StatusNotFound, "NOT_FOUND", "The requested resource was not found")
	case errors.Is(err, model.ErrDuplicate):
		RespondError(c, http.StatusConflict, "DUPLICATE", "A resource with the same identifier already exists")
	case errors.Is(err, model.ErrConflict):
		RespondError(c, http.StatusConflict, "CONFLICT", "The request conflicts with the current state")
	case errors.Is(err, model.ErrForbidden):
		RespondError(c, http.StatusForbidden, "FORBIDDEN", "You do not have permission to perform this action")
	case errors.Is(err, model.ErrInvalidInput):
		RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "The request contains invalid input")
	case errors.Is(err, model.ErrFileTooLarge):
		RespondError(c, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "The file exceeds the maximum upload size")
	case errors.Is(err, model.ErrRateLimited):
		RespondError(c, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests. Please try again later.")
	case errors.Is(err, model.ErrTokenExpired):
		RespondError(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "The token has expired")
	case errors.Is(err, model.ErrTokenRevoked):
		RespondError(c, http.StatusUnauthorized, "TOKEN_REVOKED", "The token has been revoked")
	case errors.Is(err, model.ErrUnauthenticated):
		RespondError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Invalid email or password")
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred")
	}
}

func RespondStream(c *gin.Context, status int, contentType string, reader io.Reader) {
	c.Status(status)
	c.Header("Content-Type", contentType)
	io.Copy(c.Writer, reader)
}
