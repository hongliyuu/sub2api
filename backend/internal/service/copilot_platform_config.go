package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// ErrCopilotPlatformConfigNotFound 是指定 plan_type 记录不存在时返回的错误，映射到 HTTP 404。
var ErrCopilotPlatformConfigNotFound = infraerrors.NotFound(
	"COPILOT_PLATFORM_CONFIG_NOT_FOUND",
	"copilot platform config not found",
)

// CopilotPlanType 枚举 Copilot plan 类型。
type CopilotPlanType = string

const (
	CopilotPlanIndividualFree    CopilotPlanType = "individual_free"
	CopilotPlanIndividualPro     CopilotPlanType = "individual_pro"
	CopilotPlanIndividualProPlus CopilotPlanType = "individual_pro_plus"
	CopilotPlanBusiness          CopilotPlanType = "business"
	CopilotPlanEnterprise        CopilotPlanType = "enterprise"
)

// AllCopilotPlanTypes 返回所有合法 plan_type，按展示顺序排列。
var AllCopilotPlanTypes = []CopilotPlanType{
	CopilotPlanIndividualFree,
	CopilotPlanIndividualPro,
	CopilotPlanIndividualProPlus,
	CopilotPlanBusiness,
	CopilotPlanEnterprise,
}

// CopilotPlatformConfigEntry 对应 copilot_platform_configs 表的一行。
// 所有配置字段均为指针类型，nil 表示"未设置，继承系统默认"。
type CopilotPlatformConfigEntry struct {
	ID              int64
	PlanType        string
	MaxOutputTokens *int64
	MaxBodyKB       *int
	ModelMapping    map[string]string // nil = 未设置
	ModelWhitelist  []string          // nil = 未设置
}

// CopilotPlatformConfigRepository 是平台配置的存储接口。
type CopilotPlatformConfigRepository interface {
	// GetAll 返回所有 plan_type 的配置（固定 5 行）。
	GetAll(ctx context.Context) ([]CopilotPlatformConfigEntry, error)
	// GetByPlanType 按 plan_type 查询单条记录。
	GetByPlanType(ctx context.Context, planType string) (*CopilotPlatformConfigEntry, error)
	// Upsert 更新指定 plan_type 的配置（行始终存在，只做 UPDATE）。
	Upsert(ctx context.Context, planType string, patch CopilotPlatformConfigPatch) (*CopilotPlatformConfigEntry, error)
}

// CopilotPlatformConfigPatch 描述一次 PUT 操作中要更新的字段。
type CopilotPlatformConfigPatch struct {
	MaxOutputTokens *int64
	MaxBodyKB       *int
	ModelMapping    map[string]string // nil = 清除
	ModelWhitelist  []string          // nil = 清除
	// 使用 bool 标记哪些字段被显式传入（区分"未传"和"传 null"）
	SetMaxOutputTokens bool
	SetMaxBodyKB       bool
	SetModelMapping    bool
	SetModelWhitelist  bool
}
