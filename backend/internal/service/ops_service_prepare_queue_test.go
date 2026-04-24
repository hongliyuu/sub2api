package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareOpsRequestBodyForQueue_EmptyBody(t *testing.T) {
	requestBodyJSON, truncated, requestBodyBytes := PrepareOpsRequestBodyForQueue(nil)
	require.Nil(t, requestBodyJSON)
	require.False(t, truncated)
	require.Nil(t, requestBodyBytes)
}

func TestPrepareOpsRequestBodyForQueue_InvalidJSON(t *testing.T) {
	raw := []byte("{invalid-json")
	requestBodyJSON, truncated, requestBodyBytes := PrepareOpsRequestBodyForQueue(raw)
	require.Nil(t, requestBodyJSON)
	require.False(t, truncated)
	require.NotNil(t, requestBodyBytes)
	require.Equal(t, len(raw), *requestBodyBytes)
}

func TestPrepareOpsRequestBodyForQueue_RedactSensitiveFields(t *testing.T) {
	raw := []byte(`{
		"model":"claude-3-5-sonnet-20241022",
		"api_key":"sk-test-123",
		"headers":{"authorization":"Bearer secret-token"},
		"messages":[{"role":"user","content":"hello"}]
	}`)

	requestBodyJSON, truncated, requestBodyBytes := PrepareOpsRequestBodyForQueue(raw)
	require.NotNil(t, requestBodyJSON)
	require.NotNil(t, requestBodyBytes)
	require.False(t, truncated)
	require.Equal(t, len(raw), *requestBodyBytes)

	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(*requestBodyJSON), &body))
	require.Equal(t, "[REDACTED]", body["api_key"])
	headers, ok := body["headers"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "[REDACTED]", headers["authorization"])
}

func TestPrepareOpsRequestBodyForQueue_PreservesJSONSchemaProperties(t *testing.T) {
	raw := []byte(`{
		"model":"gpt-5.5",
		"tools":[{
			"type":"function",
			"name":"write_stdin",
			"parameters":{
				"type":"object",
				"required":["session_id"],
				"properties":{
					"session_id":{"type":"number","description":"Tool session id"},
					"access_token":{"type":"string","description":"Schema property name, not a credential value"}
				}
			}
		}],
		"access_token":"real-secret"
	}`)

	requestBodyJSON, truncated, requestBodyBytes := PrepareOpsRequestBodyForQueue(raw)
	require.NotNil(t, requestBodyJSON)
	require.NotNil(t, requestBodyBytes)
	require.False(t, truncated)

	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(*requestBodyJSON), &body))
	require.Equal(t, "[REDACTED]", body["access_token"])

	tools, ok := body["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	tool, ok := tools[0].(map[string]any)
	require.True(t, ok)
	parameters, ok := tool["parameters"].(map[string]any)
	require.True(t, ok)
	properties, ok := parameters["properties"].(map[string]any)
	require.True(t, ok)
	sessionSchema, ok := properties["session_id"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "number", sessionSchema["type"])
	tokenSchema, ok := properties["access_token"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "string", tokenSchema["type"])
}

func TestPrepareOpsRequestBodyForQueue_LargeBodyTruncated(t *testing.T) {
	largeMsg := strings.Repeat("x", opsMaxStoredRequestBodyBytes*2)
	raw := []byte(`{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"` + largeMsg + `"}]}`)

	requestBodyJSON, truncated, requestBodyBytes := PrepareOpsRequestBodyForQueue(raw)
	require.NotNil(t, requestBodyJSON)
	require.NotNil(t, requestBodyBytes)
	require.True(t, truncated)
	require.Equal(t, len(raw), *requestBodyBytes)
	require.LessOrEqual(t, len(*requestBodyJSON), opsMaxStoredRequestBodyBytes)
	require.Contains(t, *requestBodyJSON, "request_body_truncated")
}
