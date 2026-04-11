package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
)

const (
	vertexGeminiDefaultLocation = "us-central1"
	vertexGeminiGlobalLocation  = "global"
	vertexGeminiDefaultVersion  = "v1beta1"
	vertexGeminiCloudScope      = "https://www.googleapis.com/auth/cloud-platform"
	vertexJWTGrantType          = "urn:ietf:params:oauth:grant-type:jwt-bearer"
)

type vertexServiceAccountCredentials struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	TokenURI     string `json:"token_uri"`
}

var vertexLocationPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

func parseVertexServiceAccountCredentials(raw string) (*vertexServiceAccountCredentials, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("vertex service_account_json not configured")
	}

	var creds vertexServiceAccountCredentials
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil, fmt.Errorf("parse vertex service_account_json: %w", err)
	}
	if strings.TrimSpace(creds.ClientEmail) == "" {
		return nil, fmt.Errorf("vertex service account client_email is required")
	}
	if strings.TrimSpace(creds.PrivateKey) == "" {
		return nil, fmt.Errorf("vertex service account private_key is required")
	}
	// 固定使用 Google 官方 token endpoint，避免把 service_account_json 中的 token_uri 作为可控 SSRF 入口。
	creds.TokenURI = geminicli.TokenURL
	return &creds, nil
}

func resolveVertexProjectID(account *Account) (string, error) {
	if account == nil {
		return "", fmt.Errorf("vertex account is nil")
	}

	if projectID := strings.TrimSpace(account.GetCredential("project_id")); projectID != "" {
		return projectID, nil
	}

	creds, err := parseVertexServiceAccountCredentials(account.GetCredential("service_account_json"))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(creds.ProjectID) == "" {
		return "", fmt.Errorf("vertex project_id not configured")
	}
	return strings.TrimSpace(creds.ProjectID), nil
}

func resolveVertexLocation(account *Account) (string, error) {
	if account == nil {
		return vertexGeminiDefaultLocation, nil
	}
	location := strings.TrimSpace(account.GetCredential("location"))
	if location == "" {
		location = vertexGeminiDefaultLocation
	}
	location = strings.ToLower(location)
	if !vertexLocationPattern.MatchString(location) {
		return "", fmt.Errorf("invalid vertex location")
	}
	return location, nil
}

func vertexEndpointBaseURL(location string) string {
	if strings.EqualFold(strings.TrimSpace(location), vertexGeminiGlobalLocation) {
		return "https://aiplatform.googleapis.com"
	}
	return fmt.Sprintf("https://%s-aiplatform.googleapis.com", location)
}

func buildGeminiVertexURL(account *Account, model, action string, stream bool) (string, error) {
	projectID, err := resolveVertexProjectID(account)
	if err != nil {
		return "", err
	}
	location, err := resolveVertexLocation(account)
	if err != nil {
		return "", err
	}
	modelID := normalizeModelNameForPricing(model)
	if strings.TrimSpace(modelID) == "" {
		return "", fmt.Errorf("vertex model is required")
	}

	baseURL := vertexEndpointBaseURL(location)
	fullURL := fmt.Sprintf(
		"%s/%s/projects/%s/locations/%s/publishers/google/models/%s:%s",
		baseURL,
		vertexGeminiDefaultVersion,
		url.PathEscape(projectID),
		url.PathEscape(location),
		url.PathEscape(modelID),
		action,
	)
	if stream {
		fullURL += "?alt=sse"
	}
	return fullURL, nil
}

func rewriteGeminiVertexGetURL(account *Account, path string) (string, error) {
	projectID, err := resolveVertexProjectID(account)
	if err != nil {
		return "", err
	}
	location, err := resolveVertexLocation(account)
	if err != nil {
		return "", err
	}
	baseURL := vertexEndpointBaseURL(location)

	path = strings.TrimSpace(path)
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("invalid path")
	}

	pathOnly := path
	query := ""
	if idx := strings.Index(path, "?"); idx >= 0 {
		pathOnly = path[:idx]
		query = path[idx:]
	}

	var suffix string
	switch {
	case pathOnly == "/v1/models" || strings.HasPrefix(pathOnly, "/v1/models/"):
		suffix = strings.TrimPrefix(pathOnly, "/v1/models")
	case pathOnly == "/v1beta/models" || strings.HasPrefix(pathOnly, "/v1beta/models/"):
		suffix = strings.TrimPrefix(pathOnly, "/v1beta/models")
	default:
		return baseURL + path, nil
	}

	return fmt.Sprintf(
		"%s/%s/projects/%s/locations/%s/publishers/google/models%s%s",
		baseURL,
		vertexGeminiDefaultVersion,
		url.PathEscape(projectID),
		url.PathEscape(location),
		suffix,
		query,
	), nil
}
