package service

import (
	"context"
	"os"
	"slices"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	WorkerJobCapabilityTextBasic       = "text.basic"
	WorkerJobCapabilityImageGeneration = "image.generation"

	WorkerJobStatusSucceeded = "succeeded"
	WorkerExecutorPyWorker   = "py-worker"

	WorkerJobsTokenHeader = "X-Sub2API-Worker-Token"
	workerJobsTokenEnv    = "SUB2API_WORKER_SHARED_TOKEN"
)

var workerSupportedCapabilities = []string{
	WorkerJobCapabilityTextBasic,
	WorkerJobCapabilityImageGeneration,
}

type WorkerJobExecuteInput struct {
	JobID       string            `json:"job_id" binding:"required"`
	Capability  string            `json:"capability" binding:"required"`
	Input       map[string]any    `json:"input,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	RequestedBy string            `json:"requested_by,omitempty"`
}

type WorkerJobExecuteResult struct {
	JobID      string            `json:"job_id"`
	Status     string            `json:"status"`
	Executor   string            `json:"executor"`
	Capability string            `json:"capability"`
	Result     any               `json:"result,omitempty"`
	Error      string            `json:"error,omitempty"`
	StartedAt  time.Time         `json:"started_at"`
	FinishedAt time.Time         `json:"finished_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type WorkerHealthStatus struct {
	Status       string   `json:"status"`
	Executor     string   `json:"executor"`
	Capabilities []string `json:"capabilities"`
	AuthHeader   string   `json:"auth_header"`
}

type WorkerJobsService struct {
	sharedToken string
}

func NewWorkerJobsService() *WorkerJobsService {
	return &WorkerJobsService{
		sharedToken: strings.TrimSpace(os.Getenv(workerJobsTokenEnv)),
	}
}

func (s *WorkerJobsService) VerifyToken(token string) error {
	if s.sharedToken == "" {
		return infraerrors.ServiceUnavailable("WORKER_AUTH_NOT_CONFIGURED", "worker shared token is not configured")
	}
	if strings.TrimSpace(token) == "" {
		return infraerrors.Unauthorized("WORKER_AUTH_REQUIRED", "worker token is required")
	}
	if strings.TrimSpace(token) != s.sharedToken {
		return infraerrors.Unauthorized("WORKER_AUTH_INVALID", "worker token is invalid")
	}
	return nil
}

func (s *WorkerJobsService) Execute(_ context.Context, input WorkerJobExecuteInput) (WorkerJobExecuteResult, error) {
	capability := strings.TrimSpace(input.Capability)
	if capability == "" {
		return WorkerJobExecuteResult{}, infraerrors.BadRequest("WORKER_JOB_CAPABILITY_REQUIRED", "capability is required")
	}
	if !slices.Contains(workerSupportedCapabilities, capability) {
		return WorkerJobExecuteResult{}, infraerrors.BadRequest("WORKER_JOB_CAPABILITY_UNSUPPORTED", "capability is not supported by worker")
	}

	now := time.Now().UTC()
	return WorkerJobExecuteResult{
		JobID:      strings.TrimSpace(input.JobID),
		Status:     WorkerJobStatusSucceeded,
		Executor:   WorkerExecutorPyWorker,
		Capability: capability,
		Result: map[string]any{
			"handled_by":   WorkerExecutorPyWorker,
			"mode":         "pilot_worker_executor",
			"capability":   capability,
			"job_id":       strings.TrimSpace(input.JobID),
			"input":        cloneWorkerAnyMap(input.Input),
			"metadata":     cloneWorkerStringMap(input.Metadata),
			"requested_by": strings.TrimSpace(input.RequestedBy),
			"description":  "private worker pilot executor completed locally",
		},
		StartedAt:  now,
		FinishedAt: now,
		Metadata:   cloneWorkerStringMap(input.Metadata),
	}, nil
}

func (s *WorkerJobsService) Health(_ context.Context) (WorkerHealthStatus, error) {
	capabilities := append([]string(nil), workerSupportedCapabilities...)
	return WorkerHealthStatus{
		Status:       "ok",
		Executor:     WorkerExecutorPyWorker,
		Capabilities: capabilities,
		AuthHeader:   WorkerJobsTokenHeader,
	}, nil
}

func cloneWorkerAnyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneWorkerStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
