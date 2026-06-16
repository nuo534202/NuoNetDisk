package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nuonuo/nuonetdisk/internal/model"
	"github.com/nuonuo/nuonetdisk/pkg/httputil"
)

func TestParseUUID(t *testing.T) {
	valid := uuid.New().String()
	parsed, err := httputil.ParseUUID(valid)
	if err != nil {
		t.Fatalf("ParseUUID with valid UUID failed: %v", err)
	}
	if parsed.String() != valid {
		t.Errorf("expected %q, got %q", valid, parsed.String())
	}

	_, err = httputil.ParseUUID("not-a-uuid")
	if err == nil {
		t.Fatal("ParseUUID with invalid string should have failed")
	}

	_, err = httputil.ParseUUID("")
	if err == nil {
		t.Fatal("ParseUUID with empty string should have failed")
	}
}

func TestParsePaginationDefaults(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	params := httputil.ParsePagination(c)
	if params.Offset != 0 {
		t.Errorf("expected Offset 0, got %d", params.Offset)
	}
	if params.Limit != 20 {
		t.Errorf("expected Limit 20, got %d", params.Limit)
	}
}

func TestParsePaginationCustom(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?offset=10&limit=50", nil)

	params := httputil.ParsePagination(c)
	if params.Offset != 10 {
		t.Errorf("expected Offset 10, got %d", params.Offset)
	}
	if params.Limit != 50 {
		t.Errorf("expected Limit 50, got %d", params.Limit)
	}
}

func TestParsePaginationMaxLimit(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?offset=0&limit=999", nil)

	params := httputil.ParsePagination(c)
	if params.Limit != 100 {
		t.Errorf("expected Limit capped at 100, got %d", params.Limit)
	}
}

func TestParsePaginationNegativeOffset(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?offset=-1&limit=20", nil)

	params := httputil.ParsePagination(c)
	if params.Offset != 0 {
		t.Errorf("expected Offset 0 for negative input, got %d", params.Offset)
	}
}

func TestParsePaginationInvalidValues(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?offset=abc&limit=xyz", nil)

	params := httputil.ParsePagination(c)
	if params.Offset != 0 {
		t.Errorf("expected Offset 0 for invalid input, got %d", params.Offset)
	}
	if params.Limit != 20 {
		t.Errorf("expected Limit 20 for invalid input, got %d", params.Limit)
	}
}

func TestRespondServiceErrorNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httputil.RespondServiceError(c, model.ErrNotFound)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestRespondServiceErrorForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httputil.RespondServiceError(c, model.ErrForbidden)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestRespondServiceErrorConflict(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httputil.RespondServiceError(c, model.ErrDuplicate)
	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
	}
}

func TestRespondServiceErrorInvalidInput(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httputil.RespondServiceError(c, model.ErrInvalidInput)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRespondServiceErrorFileTooLarge(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httputil.RespondServiceError(c, model.ErrFileTooLarge)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, w.Code)
	}
}

func TestRespondServiceErrorNil(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httputil.RespondServiceError(c, nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d for nil error (default), got %d", http.StatusOK, w.Code)
	}
}

func TestRespondServiceErrorUnauthenticated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httputil.RespondServiceError(c, model.ErrUnauthenticated)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestRespondServiceErrorUnknown(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	httputil.RespondServiceError(c, model.ErrRateLimited)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}
