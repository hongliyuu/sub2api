// backend/internal/service/anomaly_service.go
package service

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

// AnomalyType enumerates the anomaly categories.
type AnomalyType string

const (
	AnomalyZeroToken   AnomalyType = "zero_token"
	AnomalySlowRequest AnomalyType = "slow_request"
	AnomalyTimeout     AnomalyType = "timeout"
	AnomalyError       AnomalyType = "error"
)

// AnomalySettings holds admin-configurable thresholds for anomaly detection.
type AnomalySettings struct {
	SlowRequestThresholdMs int64 `json:"slow_request_threshold_ms"`
	TimeoutThresholdMs     int64 `json:"timeout_threshold_ms"`
	DetectZeroToken        bool  `json:"detect_zero_token"`
	SaveRawData            bool  `json:"save_raw_data"`
}

// defaultAnomalySettings are used when DB is unreachable.
var defaultAnomalySettings = AnomalySettings{
	SlowRequestThresholdMs: 20000,
	TimeoutThresholdMs:     60000,
	DetectZeroToken:        true,
	SaveRawData:            true,
}

const (
	settingKeySlowRequestMs   = "ops.anomaly.slow_request_threshold_ms"
	settingKeyTimeoutMs       = "ops.anomaly.timeout_threshold_ms"
	settingKeyDetectZeroToken = "ops.anomaly.detect_zero_token"
	settingKeySaveRawData     = "ops.anomaly.save_raw_data"
	anomalySettingsCacheTTL   = 30 * time.Second
)

// RequestLogInput is the data written to request_logs for an anomalous request.
type RequestLogInput struct {
	RequestID            string
	UsageLogID           *int64
	UserID               *int64
	APIKeyID             *int64
	AccountID            *int64
	GroupID              *int64
	AnomalyTypes         []string
	RequestBody          []byte
	UpstreamRequestBody  []byte
	UpstreamResponseBody []byte
}

// RequestLogData is what the repository returns when reading a request log.
type RequestLogData struct {
	RequestID            string
	AnomalyTypes         []string
	RequestBody          []byte
	UpstreamRequestBody  []byte
	UpstreamResponseBody []byte
}

// RequestLogRepository is the port for request log persistence.
type RequestLogRepository interface {
	Save(ctx context.Context, input *RequestLogInput) error
	GetByRequestID(ctx context.Context, requestID string) (*RequestLogData, error)
}

// AnomalyService handles anomaly settings (with cache) and async write of anomalous request logs.
type AnomalyService struct {
	settingRepo    SettingRepository
	requestLogRepo RequestLogRepository

	mu       sync.RWMutex
	cached   *AnomalySettings
	expireAt time.Time
}

// NewAnomalyService creates a new AnomalyService.
func NewAnomalyService(settingRepo SettingRepository, requestLogRepo RequestLogRepository) *AnomalyService {
	return &AnomalyService{
		settingRepo:    settingRepo,
		requestLogRepo: requestLogRepo,
	}
}

// GetSettings returns cached anomaly settings, refreshing from DB if TTL has expired.
// The returned pointer is a defensive copy — safe to read, must not be mutated.
func (s *AnomalyService) GetSettings(ctx context.Context) *AnomalySettings {
	s.mu.RLock()
	if s.cached != nil && time.Now().Before(s.expireAt) {
		cp := *s.cached
		s.mu.RUnlock()
		return &cp
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	// Double-check after acquiring write lock.
	if s.cached != nil && time.Now().Before(s.expireAt) {
		cp := *s.cached
		return &cp
	}

	settings := s.loadFromDB(ctx)
	s.cached = settings
	s.expireAt = time.Now().Add(anomalySettingsCacheTTL)
	cp := *settings
	return &cp
}

// loadFromDB reads settings from the DB; returns defaults on error.
func (s *AnomalyService) loadFromDB(ctx context.Context) *AnomalySettings {
	if s.settingRepo == nil {
		def := defaultAnomalySettings
		return &def
	}
	keys := []string{settingKeySlowRequestMs, settingKeyTimeoutMs, settingKeyDetectZeroToken, settingKeySaveRawData}
	vals, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		def := defaultAnomalySettings
		return &def
	}
	out := defaultAnomalySettings
	if v, ok := vals[settingKeySlowRequestMs]; ok && v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			out.SlowRequestThresholdMs = n
		}
	}
	if v, ok := vals[settingKeyTimeoutMs]; ok && v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			out.TimeoutThresholdMs = n
		}
	}
	if v, ok := vals[settingKeyDetectZeroToken]; ok && v != "" {
		out.DetectZeroToken = v == "true" || v == "1"
	}
	if v, ok := vals[settingKeySaveRawData]; ok && v != "" {
		out.SaveRawData = v == "true" || v == "1"
	}
	return &out
}

// UpdateSettings persists new settings and invalidates the local cache.
func (s *AnomalyService) UpdateSettings(ctx context.Context, settings *AnomalySettings) error {
	kvs := map[string]string{
		settingKeySlowRequestMs:   strconv.FormatInt(settings.SlowRequestThresholdMs, 10),
		settingKeyTimeoutMs:       strconv.FormatInt(settings.TimeoutThresholdMs, 10),
		settingKeyDetectZeroToken: strconv.FormatBool(settings.DetectZeroToken),
		settingKeySaveRawData:     strconv.FormatBool(settings.SaveRawData),
	}
	if err := s.settingRepo.SetMultiple(ctx, kvs); err != nil {
		return err
	}
	s.mu.Lock()
	s.cached = nil
	s.mu.Unlock()
	return nil
}

// detectAnomalies is a pure function that computes which anomaly types apply for a completed request.
func detectAnomalies(inputTokens, outputTokens int, durationMs int64, statusCode int, settings *AnomalySettings) []AnomalyType {
	var types []AnomalyType

	if settings.DetectZeroToken && inputTokens == 0 && outputTokens == 0 {
		types = append(types, AnomalyZeroToken)
	}

	if durationMs > settings.TimeoutThresholdMs {
		types = append(types, AnomalyTimeout)
	} else if durationMs > settings.SlowRequestThresholdMs {
		types = append(types, AnomalySlowRequest)
	}

	if statusCode >= 500 {
		types = append(types, AnomalyError)
	}

	return types
}

// WriteAnomalyLog detects anomalies and writes to request_logs if any are found.
// It detaches from the caller's context so that HTTP request cancellation does not
// abort the background settings read or the DB write.
// MUST be called in a goroutine — this function performs DB I/O.
func (s *AnomalyService) WriteAnomalyLog(
	ctx context.Context,
	inputTokens, outputTokens int,
	durationMs int64,
	statusCode int,
	input *RequestLogInput,
) {
	// Detach from request lifecycle: the request may already be cancelled by the
	// time this goroutine runs.  A 30-second deadline is a reasonable safety net.
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	settings := s.GetSettings(bgCtx)
	anomalies := detectAnomalies(inputTokens, outputTokens, durationMs, statusCode, settings)
	if len(anomalies) == 0 {
		return
	}

	// Work on a copy to avoid mutating the caller's struct.
	logInput := *input
	logInput.AnomalyTypes = make([]string, len(anomalies))
	for i, a := range anomalies {
		logInput.AnomalyTypes[i] = string(a)
	}

	if !settings.SaveRawData {
		logInput.RequestBody = nil
		logInput.UpstreamRequestBody = nil
		logInput.UpstreamResponseBody = nil
	}

	if s.requestLogRepo == nil {
		return
	}

	if err := s.requestLogRepo.Save(bgCtx, &logInput); err != nil {
		slog.Error("failed to write anomaly request log", "request_id", logInput.RequestID, "error", err)
	}
}
