package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestWorkerJobsServiceVerifyTokenRequiresConfig(t *testing.T) {
	t.Setenv(workerJobsTokenEnv, "")
	svc := NewWorkerJobsService()

	err := svc.VerifyToken("secret")
	require.Error(t, err)
	require.Equal(t, "WORKER_AUTH_NOT_CONFIGURED", infraerrors.Reason(err))
}

func TestWorkerJobsServiceExecuteSupportsPilotCapabilities(t *testing.T) {
	t.Setenv(workerJobsTokenEnv, "secret")
	svc := NewWorkerJobsService()

	result, err := svc.Execute(context.Background(), WorkerJobExecuteInput{
		JobID:      "job-123",
		Capability: WorkerJobCapabilityImageGeneration,
		Input: map[string]any{
			"prompt": "draw a cat",
		},
		Metadata: map[string]string{
			"origin": "center",
		},
		RequestedBy: "center",
	})
	require.NoError(t, err)
	require.Equal(t, WorkerExecutorPyWorker, result.Executor)
	require.Equal(t, WorkerJobStatusSucceeded, result.Status)
	require.Equal(t, WorkerJobCapabilityImageGeneration, result.Capability)
	require.NotNil(t, result.Result)
}

func TestWorkerJobsServiceExecuteRejectsUnsupportedCapability(t *testing.T) {
	t.Setenv(workerJobsTokenEnv, "secret")
	svc := NewWorkerJobsService()

	_, err := svc.Execute(context.Background(), WorkerJobExecuteInput{
		JobID:      "job-unsupported",
		Capability: "audio.transcription",
	})
	require.Error(t, err)
	require.Equal(t, "WORKER_JOB_CAPABILITY_UNSUPPORTED", infraerrors.Reason(err))
}
