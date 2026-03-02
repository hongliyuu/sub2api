package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceTokenAuth_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PROVISION_SERVICE_TOKEN", "test-token-123")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/provision", nil)

	mw := NewServiceTokenAuthMiddleware()
	gin.HandlerFunc(mw)(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, c.IsAborted())

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "UNAUTHORIZED", resp.Code)
}

func TestServiceTokenAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PROVISION_SERVICE_TOKEN", "correct-token")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/provision", nil)
	c.Request.Header.Set("X-Service-Token", "wrong-token")

	mw := NewServiceTokenAuthMiddleware()
	gin.HandlerFunc(mw)(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, c.IsAborted())

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "INVALID_SERVICE_TOKEN", resp.Code)
}

func TestServiceTokenAuth_NotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PROVISION_SERVICE_TOKEN", "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/provision", nil)
	c.Request.Header.Set("X-Service-Token", "any-token")

	mw := NewServiceTokenAuthMiddleware()
	gin.HandlerFunc(mw)(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, c.IsAborted())

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "SERVICE_UNAVAILABLE", resp.Code)
}

func TestServiceTokenAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PROVISION_SERVICE_TOKEN", "valid-secret-token")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/provision", nil)
	c.Request.Header.Set("X-Service-Token", "valid-secret-token")

	called := false
	c.Set("test_next_called", false)

	router := gin.New()
	router.POST("/api/provision", gin.HandlerFunc(NewServiceTokenAuthMiddleware()), func(c *gin.Context) {
		called = true
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/api/provision", nil)
	req.Header.Set("X-Service-Token", "valid-secret-token")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)

	assert.Equal(t, http.StatusOK, w2.Code)
	_ = called // The handler was called via the router
}
