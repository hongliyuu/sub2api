// backend/ent/schema/copilot_platform_config.go
package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CopilotPlatformConfig holds the schema definition for Copilot platform-level defaults.
//
// 存储按 plan_type 分组的平台级默认参数。
// 账号级配置（credentials 字段）优先；账号未设置时继承此处配置；两者都没有时使用系统默认。
type CopilotPlatformConfig struct {
	ent.Schema
}

func (CopilotPlatformConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "copilot_platform_configs"},
	}
}

func (CopilotPlatformConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (CopilotPlatformConfig) Fields() []ent.Field {
	return []ent.Field{
		// 枚举值: individual_free / individual_pro / individual_pro_plus / business / enterprise
		field.String("plan_type").
			MaxLen(32).
			NotEmpty().
			Unique().
			Comment("Copilot plan type: individual_free / individual_pro / individual_pro_plus / business / enterprise"),

		// NULL 表示该 plan_type 不设默认值
		field.Int64("max_output_tokens").
			Optional().
			Nillable().
			Comment("Max output tokens for this plan type; NULL = use system default"),

		field.Int("max_body_kb").
			Optional().
			Nillable().
			Comment("Max request body size in KB for this plan type; NULL = use system default"),

		// JSONB 存储为 map[string]string
		field.JSON("model_mapping", map[string]string{}).
			Optional().
			Comment("Model name rewriting map {from: to}; NULL = no mapping default"),

		// JSONB 存储为 []string
		field.JSON("model_whitelist", []string{}).
			Optional().
			Comment("Allowed model list; NULL = allow all (no whitelist default)"),
	}
}

func (CopilotPlatformConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("plan_type").Unique(),
	}
}
