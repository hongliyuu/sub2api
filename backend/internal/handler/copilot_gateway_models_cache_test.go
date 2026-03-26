package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestCopilotHandler() *CopilotGatewayHandler {
	return &CopilotGatewayHandler{
		modelCache: make(map[int64]*copilotModelCacheEntry),
	}
}

// ── getModelCache / setModelCache ─────────────────────────────────────────────

func TestCopilotModelCache_MissOnEmpty(t *testing.T) {
	h := newTestCopilotHandler()
	_, ok := h.getModelCache(0)
	require.False(t, ok, "empty cache should be a miss")
}

func TestCopilotModelCache_HitWhenFresh(t *testing.T) {
	h := newTestCopilotHandler()
	data := []byte(`{"object":"list","data":[]}`)
	h.setModelCache(1, data, true)

	got, ok := h.getModelCache(1)
	require.True(t, ok, "fresh upstream cache should hit")
	require.Equal(t, data, got)
}

func TestCopilotModelCache_MissWhenExpiredUpstream(t *testing.T) {
	h := newTestCopilotHandler()
	data := []byte(`{"object":"list","data":[]}`)
	h.setModelCache(1, data, true)

	// Manually back-date the entry past the upstream TTL.
	h.modelCacheMu.Lock()
	h.modelCache[1].cachedAt = time.Now().Add(-(copilotModelCacheTTL + time.Second))
	h.modelCacheMu.Unlock()

	_, ok := h.getModelCache(1)
	require.False(t, ok, "expired upstream cache should be a miss")
}

func TestCopilotModelCache_HitWhenFreshFallback(t *testing.T) {
	h := newTestCopilotHandler()
	data := []byte(`{"object":"list","data":[]}`)
	h.setModelCache(2, data, false) // static fallback

	got, ok := h.getModelCache(2)
	require.True(t, ok, "fresh fallback cache should hit")
	require.Equal(t, data, got)
}

func TestCopilotModelCache_MissWhenExpiredFallback(t *testing.T) {
	h := newTestCopilotHandler()
	data := []byte(`{"object":"list","data":[]}`)
	h.setModelCache(2, data, false)

	// Back-date past the short failed TTL.
	h.modelCacheMu.Lock()
	h.modelCache[2].cachedAt = time.Now().Add(-(copilotModelCacheFailedTTL + time.Second))
	h.modelCacheMu.Unlock()

	_, ok := h.getModelCache(2)
	require.False(t, ok, "expired fallback cache should be a miss")
}

func TestCopilotModelCache_GroupIsolation(t *testing.T) {
	h := newTestCopilotHandler()
	h.setModelCache(10, []byte(`{"group":10}`), true)
	h.setModelCache(20, []byte(`{"group":20}`), true)

	got10, ok10 := h.getModelCache(10)
	got20, ok20 := h.getModelCache(20)
	require.True(t, ok10)
	require.True(t, ok20)
	require.Equal(t, []byte(`{"group":10}`), got10)
	require.Equal(t, []byte(`{"group":20}`), got20)

	// Group 30 should still miss.
	_, ok30 := h.getModelCache(30)
	require.False(t, ok30)
}

// ── getStaleModelCache ────────────────────────────────────────────────────────

func TestCopilotModelCache_StaleReturnsExpiredEntry(t *testing.T) {
	h := newTestCopilotHandler()
	data := []byte(`{"object":"list","data":[]}`)
	h.setModelCache(5, data, true)

	// Expire the entry.
	h.modelCacheMu.Lock()
	h.modelCache[5].cachedAt = time.Now().Add(-(copilotModelCacheTTL + time.Second))
	h.modelCacheMu.Unlock()

	// getModelCache should miss but getStaleModelCache should return the data.
	_, freshOK := h.getModelCache(5)
	require.False(t, freshOK, "expired entry should be a fresh miss")

	stale := h.getStaleModelCache(5)
	require.Equal(t, data, stale, "getStaleModelCache should return the expired entry")
}

func TestCopilotModelCache_StaleReturnsNilWhenEmpty(t *testing.T) {
	h := newTestCopilotHandler()
	require.Nil(t, h.getStaleModelCache(99), "no entry should return nil")
}

// ── buildDefaultCopilotModelsJSON ─────────────────────────────────────────────

func TestBuildDefaultCopilotModelsJSON_ValidJSON(t *testing.T) {
	data := buildDefaultCopilotModelsJSON()
	require.NotEmpty(t, data)

	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result), "output must be valid JSON")

	require.Equal(t, "list", result["object"], "object field must be 'list'")

	arr, ok := result["data"].([]any)
	require.True(t, ok, "data field must be an array")
	require.NotEmpty(t, arr, "data array must not be empty (DefaultModels has entries)")
}

func TestBuildDefaultCopilotModelsJSON_EachModelHasID(t *testing.T) {
	data := buildDefaultCopilotModelsJSON()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(data, &result))

	for i, m := range result.Data {
		require.NotEmpty(t, m.ID, "DefaultModels[%d] must have a non-empty id", i)
	}
}
