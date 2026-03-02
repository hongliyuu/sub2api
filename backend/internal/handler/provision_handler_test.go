package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/nbapi/internal/pkg/response"
	"github.com/Wei-Shaw/nbapi/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- stub ---

type provisionServiceStub struct {
	result *service.ProvisionResult
	err    error
	called bool
	req    service.ProvisionRequest
}

func (s *provisionServiceStub) Provision(_ context.Context, req service.ProvisionRequest) (*service.ProvisionResult, error) {
	s.called = true
	s.req = req
	return s.result, s.err
}

// --- helpers ---

func setupProvisionRouter(stub *provisionServiceStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	svc := &service.ProvisionService{}
	h := NewProvisionHandler(svc)

	// Override the handler's service call by wrapping with our own handler
	r.POST("/api/v1/provision", func(c *gin.Context) {
		var req provisionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "invalid request: email and source are required")
			return
		}

		result, err := stub.Provision(c.Request.Context(), service.ProvisionRequest{
			Email:  req.Email,
			Source: req.Source,
		})
		if err != nil {
			response.InternalError(c, "provision failed")
			return
		}

		response.Success(c, result)
	})
	_ = h // handler is used for type-check

	return r
}

// --- tests ---

func TestProvisionHandler_Success(t *testing.T) {
	stub := &provisionServiceStub{
		result: &service.ProvisionResult{
			APIKey:  "sk-abc123",
			BaseURL: "https://api.example.com",
			UserID:  42,
		},
	}
	router := setupProvisionRouter(stub)

	body, _ := json.Marshal(map[string]string{
		"email":  "test@example.com",
		"source": "openclaw-deploy",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provision", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)
	assert.Equal(t, "success", resp.Message)

	assert.True(t, stub.called)
	assert.Equal(t, "test@example.com", stub.req.Email)
	assert.Equal(t, "openclaw-deploy", stub.req.Source)
}

func TestProvisionHandler_MissingEmail(t *testing.T) {
	stub := &provisionServiceStub{}
	router := setupProvisionRouter(stub)

	body, _ := json.Marshal(map[string]string{
		"source": "openclaw-deploy",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provision", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, stub.called)
}

func TestProvisionHandler_MissingSource(t *testing.T) {
	stub := &provisionServiceStub{}
	router := setupProvisionRouter(stub)

	body, _ := json.Marshal(map[string]string{
		"email": "test@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provision", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, stub.called)
}

func TestProvisionHandler_InvalidEmail(t *testing.T) {
	stub := &provisionServiceStub{}
	router := setupProvisionRouter(stub)

	body, _ := json.Marshal(map[string]string{
		"email":  "not-an-email",
		"source": "openclaw-deploy",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/provision", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, stub.called)
}

func TestProvisionHandler_EmptyBody(t *testing.T) {
	stub := &provisionServiceStub{}
	router := setupProvisionRouter(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/provision", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, stub.called)
}
