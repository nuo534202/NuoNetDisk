package httputil

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

func ParseUUID(str string) (uuid.UUID, error) {
	return uuid.Parse(str)
}

type PaginationParams struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func ParsePagination(c *gin.Context) PaginationParams {
	offset := 0
	limit := defaultLimit

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
			if limit > maxLimit {
				limit = maxLimit
			}
		}
	}

	return PaginationParams{
		Offset: offset,
		Limit:  limit,
	}
}
