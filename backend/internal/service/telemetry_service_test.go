package service

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tidwall/gjson"
)

func TestDeepScrubPayload_RealSchema(t *testing.T) {
	svc := NewTelemetryService()
	rawMeta := `{
		"baseUrl":"http://sub2api.local:8080/v1/messages",
		"gateway":"sub2api",
		"safe_info":"keep_this",
		"env":{
			"platform":"linux",
			"platform_raw":"linux",
			"arch":"x64",
			"node_version":"v18.20.0",
			"terminal":"gnome-terminal",
			"package_managers":"npm,yarn",
			"runtimes":"node",
			"is_running_with_bun":false,
			"deployment_environment":"unknown-linux",
			"wsl_version":"WSL2",
			"linux_distro_id":"ubuntu",
			"linux_distro_version":"22.04",
			"linux_kernel":"5.15.0",
			"is_ci":true,
			"is_github_action":true,
			"github_actions_metadata":{"actor_id":"12345","repository_id":"67890"}
		}
	}`
	encodedMeta := base64.StdEncoding.EncodeToString([]byte(rawMeta))

	payload := `{
		"events": [
			{
				"event_type": "GrowthbookExperimentEvent",
				"event_data": {
					"event_id": "growth-event-1",
					"timestamp": "2026-04-02T08:00:00Z",
					"device_id": "windows_device_aaa",
					"session_id": "growth-session-1",
					"anonymous_id": "anon-1",
					"event_metadata_vars": "secret-meta",
					"auth": {"account_uuid":"shared-account-uuid-001","organization_uuid":"org-secret-1"},
					"user_attributes": "{\"id\":\"windows_device_aaa\",\"deviceID\":\"windows_device_aaa\",\"apiBaseUrlHost\":\"sub2api.local:8080\",\"email\":\"user1@gmail.com\",\"githubActionsMetadata\":{\"repo\":\"secret\"},\"accountUUID\":\"shared-account-uuid-001\",\"organizationUUID\":\"org-secret-1\",\"platform\":\"win32\",\"subscriptionType\":\"pro\"}"
				}
			},
			{
				"event_type": "ClaudeCodeInternalEvent",
				"event_data": {
					"event_id": "internal-event-1",
					"client_timestamp": "2026-04-02T08:01:00Z",
					"device_id": "mac_device_ccc",
					"session_id": "internal-session-1",
					"parent_session_id": "parent-session-1",
					"agent_id": "worker-abc@swarm-main",
					"process": "{\"pid\":999,\"rss\":123}",
					"email": "user3@hack.local",
					"auth": {"account_uuid":"shared-account-uuid-001","organization_uuid":"org-secret-1"},
					"env": {
						"platform":"linux",
						"platform_raw":"linux",
						"arch":"x64",
						"node_version":"v18.20.0",
						"terminal":"gnome-terminal",
						"package_managers":"npm,yarn",
						"runtimes":"node",
						"is_running_with_bun":false,
						"deployment_environment":"unknown-linux",
						"wsl_version":"WSL2",
						"linux_distro_id":"ubuntu",
						"linux_distro_version":"22.04",
						"linux_kernel":"5.15.0",
						"github_actions_metadata":{"actor_id":"12345","repository_id":"67890"},
						"is_ci":true,
						"is_github_action":true
					},
					"additional_metadata": "` + encodedMeta + `"
				}
			}
		]
	}`

	scrubbedBytes, err := svc.DeepScrubPayload([]byte(payload))
	if err != nil {
		t.Fatalf("DeepScrubPayload failed: %v", err)
	}

	result := string(scrubbedBytes)
	if strings.Contains(result, "windows_device_aaa") || strings.Contains(result, "mac_device_ccc") {
		t.Fatalf("original device_id leaked: %s", result)
	}
	if strings.Contains(result, "user1@gmail.com") || strings.Contains(result, "user3@hack.local") {
		t.Fatalf("email leaked: %s", result)
	}
	if strings.Contains(result, "org-secret-1") || strings.Contains(result, "shared-account-uuid-001") {
		t.Fatalf("auth/account identifier leaked: %s", result)
	}
	if strings.Contains(result, "sub2api") {
		t.Fatalf("gateway marker leaked: %s", result)
	}

	growthDev := gjson.GetBytes(scrubbedBytes, "events.0.event_data.device_id").String()
	internalDev := gjson.GetBytes(scrubbedBytes, "events.1.event_data.device_id").String()
	if growthDev == "" || internalDev == "" {
		t.Fatalf("shadow device ids missing")
	}
	if growthDev != internalDev {
		t.Fatalf("same account should converge to one shadow device id: %s vs %s", growthDev, internalDev)
	}

	if gjson.GetBytes(scrubbedBytes, "events.0.event_data.auth").Exists() || gjson.GetBytes(scrubbedBytes, "events.1.event_data.auth").Exists() {
		t.Fatalf("auth block should be deleted")
	}
	if gjson.GetBytes(scrubbedBytes, "events.0.event_data.event_metadata_vars").Exists() {
		t.Fatalf("event_metadata_vars should be deleted")
	}

	if got := gjson.GetBytes(scrubbedBytes, "events.0.event_data.session_id").String(); got == "growth-session-1" || got == "" {
		t.Fatalf("growthbook session_id not remapped: %q", got)
	}
	if got := gjson.GetBytes(scrubbedBytes, "events.0.event_data.anonymous_id").String(); got == "anon-1" || got == "" {
		t.Fatalf("anonymous_id not remapped: %q", got)
	}
	if got := gjson.GetBytes(scrubbedBytes, "events.1.event_data.parent_session_id").String(); got == "parent-session-1" || got == "" {
		t.Fatalf("parent_session_id not remapped: %q", got)
	}
	if got := gjson.GetBytes(scrubbedBytes, "events.1.event_data.agent_id").String(); got == "worker-abc@swarm-main" || !strings.Contains(got, "@") {
		t.Fatalf("agent_id not remapped to teammate-style shape: %q", got)
	}

	if _, err := time.Parse(time.RFC3339Nano, gjson.GetBytes(scrubbedBytes, "events.0.event_data.timestamp").String()); err != nil {
		t.Fatalf("growthbook timestamp not rewritten to RFC3339: %v", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, gjson.GetBytes(scrubbedBytes, "events.1.event_data.client_timestamp").String()); err != nil {
		t.Fatalf("client_timestamp not rewritten to RFC3339: %v", err)
	}
	timestamp := gjson.GetBytes(scrubbedBytes, "events.0.event_data.timestamp").String()
	if strings.HasSuffix(timestamp, ".000000000Z") || strings.HasSuffix(timestamp, ".000000Z") {
		t.Fatalf("timestamp still looks discretized: %q", timestamp)
	}

	userAttrsRaw := gjson.GetBytes(scrubbedBytes, "events.0.event_data.user_attributes").String()
	var userAttrs map[string]any
	if err := json.Unmarshal([]byte(userAttrsRaw), &userAttrs); err != nil {
		t.Fatalf("sanitized user_attributes is invalid json: %v", err)
	}
	if userAttrs["id"] != growthDev || userAttrs["deviceID"] != growthDev {
		t.Fatalf("user_attributes ids not rewritten: %+v", userAttrs)
	}
	if userAttrs["platform"] != "darwin" {
		t.Fatalf("user_attributes platform not overwritten: %+v", userAttrs)
	}
	if _, ok := userAttrs["subscriptionType"]; !ok {
		t.Fatalf("subscriptionType should be preserved: %+v", userAttrs)
	}
	for _, key := range []string{"email", "apiBaseUrlHost", "githubActionsMetadata", "accountUUID", "organizationUUID"} {
		if _, ok := userAttrs[key]; ok {
			t.Fatalf("user_attributes leaked %s: %+v", key, userAttrs)
		}
	}

	env := gjson.GetBytes(scrubbedBytes, "events.1.event_data.env")
	if env.Get("platform").String() != "darwin" || env.Get("platform_raw").String() != "darwin" {
		t.Fatalf("env platform not overwritten: %s", env.Raw)
	}
	if env.Get("node_version").String() == "v18.20.0" || env.Get("node_version").String() == "" {
		t.Fatalf("env node_version not overwritten: %s", env.Raw)
	}
	if env.Get("terminal").String() == "gnome-terminal" || env.Get("terminal").String() == "" {
		t.Fatalf("env terminal not diversified: %s", env.Raw)
	}
	if env.Get("package_managers").String() == "npm,yarn" || env.Get("package_managers").String() == "" {
		t.Fatalf("env package_managers not diversified: %s", env.Raw)
	}
	if env.Get("runtimes").String() == "node" || env.Get("runtimes").String() == "" {
		t.Fatalf("env runtimes not diversified: %s", env.Raw)
	}
	if env.Get("version").String() == "" || env.Get("version_base").String() == "" {
		t.Fatalf("env version fields missing: %s", env.Raw)
	}
	for _, key := range []string{"wsl_version", "linux_distro_id", "linux_distro_version", "linux_kernel", "github_actions_metadata", "vcs"} {
		if env.Get(key).Exists() {
			t.Fatalf("env leaked %s: %s", key, env.Raw)
		}
	}
	if env.Get("is_ci").Bool() || env.Get("is_github_action").Bool() || env.Get("is_claude_code_remote").Bool() {
		t.Fatalf("env boolean scrub failed: %s", env.Raw)
	}
	if env.Get("deployment_environment").String() == "unknown-linux" || env.Get("deployment_environment").String() == "" {
		t.Fatalf("deployment_environment not diversified: %s", env.Raw)
	}
	if env.Get("build_time").Exists() {
		t.Fatalf("build_time should be dropped rather than guessed: %s", env.Raw)
	}
	processRaw := gjson.GetBytes(scrubbedBytes, "events.1.event_data.process").String()
	if processRaw == "" {
		t.Fatalf("process metrics missing")
	}
	var processMetrics map[string]any
	if err := json.Unmarshal([]byte(processRaw), &processMetrics); err != nil {
		t.Fatalf("process metrics invalid json: %v", err)
	}
	for _, key := range []string{"uptime", "rss", "heapTotal", "heapUsed", "external", "arrayBuffers", "constrainedMemory", "cpuUsage", "cpuPercent"} {
		if _, ok := processMetrics[key]; !ok {
			t.Fatalf("process metrics missing %s: %+v", key, processMetrics)
		}
	}
	if processMetrics["uptime"].(float64) <= 0 {
		t.Fatalf("process uptime must be positive: %+v", processMetrics)
	}
	cpuUsage, ok := processMetrics["cpuUsage"].(map[string]any)
	if !ok || cpuUsage["user"] == nil || cpuUsage["system"] == nil {
		t.Fatalf("cpuUsage missing expected shape: %+v", processMetrics)
	}

	metaB64 := gjson.GetBytes(scrubbedBytes, "events.1.event_data.additional_metadata").String()
	metaBytes, err := base64.StdEncoding.DecodeString(metaB64)
	if err != nil {
		t.Fatalf("additional_metadata is not valid base64: %v", err)
	}
	meta := gjson.ParseBytes(metaBytes)
	if meta.Get("safe_info").String() != "keep_this" {
		t.Fatalf("safe_info should be preserved: %s", meta.Raw)
	}
	if meta.Get("baseUrl").Exists() || meta.Get("gateway").Exists() {
		t.Fatalf("additional_metadata leaked gateway fields: %s", meta.Raw)
	}
	if meta.Get("env.platform").String() != "darwin" {
		t.Fatalf("additional_metadata env not scrubbed: %s", meta.Raw)
	}
	if meta.Get("env.version").String() == "" || meta.Get("env.version_base").String() == "" {
		t.Fatalf("additional_metadata env version fields missing: %s", meta.Raw)
	}
	if meta.Get("env.build_time").Exists() {
		t.Fatalf("additional_metadata build_time should be dropped rather than guessed: %s", meta.Raw)
	}
}

func TestDeepScrubPayload_EmptyAndMalformed(t *testing.T) {
	svc := NewTelemetryService()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "malformed json", input: `{not valid json`, wantErr: true},
		{name: "no events field", input: `{"foo":"bar"}`, wantErr: true},
		{name: "events is not array", input: `{"events":"not_an_array"}`, wantErr: true},
		{name: "empty events array", input: `{"events":[]}`},
		{name: "event without event_data", input: `{"events":[{"event_type":"GrowthbookExperimentEvent"}]}`, wantErr: true},
		{name: "event with non-object event_data", input: `{"events":[{"event_type":"GrowthbookExperimentEvent","event_data":"string_not_map"}]}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.DeepScrubPayload([]byte(tt.input))
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDeepScrubPayload_InvalidBase64Metadata(t *testing.T) {
	svc := NewTelemetryService()
	payload := `{
		"events": [{
			"event_type": "ClaudeCodeInternalEvent",
			"event_data": {
				"device_id": "dev-123",
				"client_timestamp": "2026-04-02T08:01:00Z",
				"additional_metadata": "NOT_VALID_BASE64!!!"
			}
		}]
	}`

	result, err := svc.DeepScrubPayload([]byte(payload))
	if err != nil {
		t.Fatalf("should not error on invalid base64 metadata: %v", err)
	}
	if strings.Contains(string(result), "dev-123") {
		t.Fatalf("original device_id leaked despite bad metadata")
	}
	if gjson.GetBytes(result, "events.0.event_data.additional_metadata").Exists() {
		t.Fatalf("invalid additional_metadata should be deleted")
	}
}

func TestDeepScrubPayload_InvalidUserAttributes(t *testing.T) {
	svc := NewTelemetryService()
	payload := `{
		"events": [{
			"event_type": "GrowthbookExperimentEvent",
			"event_data": {
				"device_id": "dev-456",
				"user_attributes": "this is {not} valid json"
			}
		}]
	}`

	result, err := svc.DeepScrubPayload([]byte(payload))
	if err != nil {
		t.Fatalf("should not error on invalid user_attributes json: %v", err)
	}
	if strings.Contains(string(result), "dev-456") {
		t.Fatalf("original device_id leaked despite bad user_attributes")
	}
	userAttrs := gjson.GetBytes(result, "events.0.event_data.user_attributes").String()
	if !strings.Contains(userAttrs, `"platform":"darwin"`) {
		t.Fatalf("invalid user_attributes should be replaced with sanitized json: %s", userAttrs)
	}
}

func TestGenerateShadowDeviceID_UUIDFormat(t *testing.T) {
	svc := NewTelemetryService()

	seeds := []string{"test-uuid-1", "another-seed", "", "shared-account-uuid-001"}
	for _, seed := range seeds {
		id := svc.GenerateShadowDeviceID(seed, "")
		parts := strings.Split(id, "-")
		if len(parts) != 5 {
			t.Fatalf("seed=%q: expected 5 parts, got %d: %s", seed, len(parts), id)
		}
		if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 || len(parts[3]) != 4 || len(parts[4]) != 12 {
			t.Fatalf("seed=%q: wrong part lengths in %s", seed, id)
		}
		if parts[2][0] != '4' {
			t.Fatalf("seed=%q: version nibble should be '4', got %q in %s", seed, parts[2][0], id)
		}
		v := parts[3][0]
		if v != '8' && v != '9' && v != 'a' && v != 'b' {
			t.Fatalf("seed=%q: variant nibble should be 8/9/a/b, got %q in %s", seed, v, id)
		}
	}

	id1 := svc.GenerateShadowDeviceID("shared-account-uuid-001", "device-a")
	id2 := svc.GenerateShadowDeviceID("shared-account-uuid-001", "device-b")
	if id1 != id2 {
		t.Fatalf("same account seed should converge despite different device ids: %s vs %s", id1, id2)
	}
}

func TestGenerateOpaqueAgentID_PreservesNativeShapes(t *testing.T) {
	svc := NewTelemetryService()

	teamStyle := svc.GenerateOpaqueAgentID("shadow-device-1", "worker-abc@swarm-main")
	if teamStyle == "worker-abc@swarm-main" || !strings.Contains(teamStyle, "@") {
		t.Fatalf("expected teammate-style agent id, got %q", teamStyle)
	}

	uuidStyle := svc.GenerateOpaqueAgentID("shadow-device-1", "8cdbd330-29f8-4b73-b4f4-cf5f0fe3d300")
	if uuidStyle == "8cdbd330-29f8-4b73-b4f4-cf5f0fe3d300" || strings.Contains(uuidStyle, "@") {
		t.Fatalf("expected uuid-style agent id, got %q", uuidStyle)
	}
	parts := strings.Split(uuidStyle, "-")
	if len(parts) != 5 {
		t.Fatalf("uuid-style agent id malformed: %q", uuidStyle)
	}
}

func TestSyntheticProcessMetrics_MonotonicPerDevice(t *testing.T) {
	syntheticProcessStore.reset()

	first := syntheticProcessMetrics("shadow-device-uptime", time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC))
	second := syntheticProcessMetrics("shadow-device-uptime", time.Date(2026, 4, 2, 10, 0, 5, 0, time.UTC))
	third := syntheticProcessMetrics("shadow-device-uptime", time.Date(2026, 4, 2, 10, 0, 4, 0, time.UTC))

	parse := func(raw string) map[string]any {
		var out map[string]any
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			t.Fatalf("invalid process json: %v", err)
		}
		return out
	}

	p1 := parse(first)
	p2 := parse(second)
	p3 := parse(third)

	if p2["uptime"].(float64) <= p1["uptime"].(float64) {
		t.Fatalf("uptime must increase: p1=%v p2=%v", p1["uptime"], p2["uptime"])
	}
	if p3["uptime"].(float64) <= p2["uptime"].(float64) {
		t.Fatalf("uptime must keep increasing even if timestamps arrive out of order: p2=%v p3=%v", p2["uptime"], p3["uptime"])
	}

	cpu1 := p1["cpuUsage"].(map[string]any)
	cpu2 := p2["cpuUsage"].(map[string]any)
	cpu3 := p3["cpuUsage"].(map[string]any)
	if cpu2["user"].(float64) <= cpu1["user"].(float64) || cpu3["user"].(float64) <= cpu2["user"].(float64) {
		t.Fatalf("cpu user counter must be monotonic: %+v %+v %+v", cpu1, cpu2, cpu3)
	}
	if cpu2["system"].(float64) <= cpu1["system"].(float64) || cpu3["system"].(float64) <= cpu2["system"].(float64) {
		t.Fatalf("cpu system counter must be monotonic: %+v %+v %+v", cpu1, cpu2, cpu3)
	}

	if p1["rss"].(float64) == p2["rss"].(float64) && p2["rss"].(float64) == p3["rss"].(float64) {
		t.Fatalf("rss should vary over time: %v %v %v", p1["rss"], p2["rss"], p3["rss"])
	}
	if p1["heapUsed"].(float64) == p2["heapUsed"].(float64) && p2["heapUsed"].(float64) == p3["heapUsed"].(float64) {
		t.Fatalf("heapUsed should vary over time: %v %v %v", p1["heapUsed"], p2["heapUsed"], p3["heapUsed"])
	}
	if p1["external"].(float64) == p2["external"].(float64) && p2["external"].(float64) == p3["external"].(float64) {
		t.Fatalf("external should vary over time: %v %v %v", p1["external"], p2["external"], p3["external"])
	}

	checkCPUPercent := func(prev, next map[string]any, deltaSeconds float64) {
		prevCPU := prev["cpuUsage"].(map[string]any)
		nextCPU := next["cpuUsage"].(map[string]any)
		deltaCPU := (nextCPU["user"].(float64) - prevCPU["user"].(float64)) + (nextCPU["system"].(float64) - prevCPU["system"].(float64))
		expected := (deltaCPU / (deltaSeconds * 1_000_000.0)) * 100.0
		reported := next["cpuPercent"].(float64)
		if math.Abs(expected-reported) > 0.2 {
			t.Fatalf("cpuPercent inconsistent: expected %.3f reported %.3f", expected, reported)
		}
	}
	checkCPUPercent(p1, p2, 5)
	checkCPUPercent(p2, p3, 1)
}

func TestProcessStateStore_PersistsAcrossRecreation(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "telemetry-process-state.json")
	t.Setenv("TELEMETRY_PROCESS_STATE_FILE", stateFile)
	t.Cleanup(func() {
		syntheticProcessStore = newProcessStateStore()
	})

	storeA := newProcessStateStore()
	syntheticProcessStore = storeA
	first := syntheticProcessMetrics("shadow-device-persist", time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC))
	second := syntheticProcessMetrics("shadow-device-persist", time.Date(2026, 4, 2, 10, 0, 10, 0, time.UTC))

	raw, err := os.ReadFile(stateFile)
	if err != nil || len(raw) == 0 {
		t.Fatalf("expected persisted process state file, err=%v", err)
	}

	storeB := newProcessStateStore()
	syntheticProcessStore = storeB
	third := syntheticProcessMetrics("shadow-device-persist", time.Date(2026, 4, 2, 10, 0, 20, 0, time.UTC))

	parse := func(raw string) map[string]any {
		var out map[string]any
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			t.Fatalf("invalid process json: %v", err)
		}
		return out
	}

	p1 := parse(first)
	p2 := parse(second)
	p3 := parse(third)
	if p2["uptime"].(float64) <= p1["uptime"].(float64) || p3["uptime"].(float64) <= p2["uptime"].(float64) {
		t.Fatalf("uptime should remain monotonic across recreated store: %v %v %v", p1["uptime"], p2["uptime"], p3["uptime"])
	}
	cpu2 := p2["cpuUsage"].(map[string]any)
	cpu3 := p3["cpuUsage"].(map[string]any)
	if cpu3["user"].(float64) <= cpu2["user"].(float64) || cpu3["system"].(float64) <= cpu2["system"].(float64) {
		t.Fatalf("cpu counters should remain monotonic across recreated store: %+v %+v", cpu2, cpu3)
	}
}

func TestFindTelemetrySidecarScript_Override(t *testing.T) {
	script := filepath.Join(t.TempDir(), "telemetry-sidecar.mjs")
	if err := os.WriteFile(script, []byte("console.log('ok')\n"), 0o600); err != nil {
		t.Fatalf("write temp sidecar: %v", err)
	}
	t.Setenv("TELEMETRY_NODE_SIDECAR_SCRIPT", script)

	got, err := findTelemetrySidecarScript()
	if err != nil {
		t.Fatalf("expected override sidecar path to resolve: %v", err)
	}
	if got != script {
		t.Fatalf("override sidecar path mismatch: got %q want %q", got, script)
	}
}

func TestForwardWithNodeSidecar_Success(t *testing.T) {
	script := filepath.Join(t.TempDir(), "telemetry-sidecar.mjs")
	code := `
const chunks = [];
for await (const chunk of process.stdin) chunks.push(chunk);
const input = JSON.parse(Buffer.concat(chunks).toString('utf8'));
if (input.headers["User-Agent"] !== "claude-code/2.2.19") {
  throw new Error("unexpected user agent");
}
if (input.headers["x-service-name"] !== "claude-code") {
  throw new Error("missing service name");
}
if (input.headers["x-api-key"] !== "test-key") {
  throw new Error("missing api key");
}
const payload = Buffer.from(input.payload_base64, 'base64').toString('utf8');
if (!payload.includes('"events"')) {
  throw new Error("missing payload");
}
process.stdout.write(JSON.stringify({status: 204}));
`
	if err := os.WriteFile(script, []byte(code), 0o600); err != nil {
		t.Fatalf("write temp sidecar: %v", err)
	}
	t.Setenv("TELEMETRY_NODE_SIDECAR_SCRIPT", script)

	svc := NewTelemetryService()
	err := svc.forwardWithNodeSidecar([]byte(`{"events":[]}`), "test-key", "https://api.anthropic.com/api/event_logging/batch", "2.2.19")
	if err != nil {
		t.Fatalf("expected sidecar success, got %v", err)
	}
}

func TestForwardWithNodeSidecar_Error(t *testing.T) {
	script := filepath.Join(t.TempDir(), "telemetry-sidecar.mjs")
	code := `
process.stdout.write(JSON.stringify({status: 0, error: "boom"}));
`
	if err := os.WriteFile(script, []byte(code), 0o600); err != nil {
		t.Fatalf("write temp sidecar: %v", err)
	}
	t.Setenv("TELEMETRY_NODE_SIDECAR_SCRIPT", script)

	svc := NewTelemetryService()
	err := svc.forwardWithNodeSidecar([]byte(`{"events":[]}`), "", "https://api.anthropic.com/api/event_logging/batch", "2.2.19")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected propagated sidecar error, got %v", err)
	}
}

func TestAllowGoFallback_DefaultOff(t *testing.T) {
	t.Setenv("TELEMETRY_ALLOW_GO_FALLBACK", "")
	if allowGoFallback() {
		t.Fatalf("expected Go fallback to be disabled by default")
	}

	t.Setenv("TELEMETRY_ALLOW_GO_FALLBACK", "true")
	if !allowGoFallback() {
		t.Fatalf("expected Go fallback to be enabled when explicitly requested")
	}
}
