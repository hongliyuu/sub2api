package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// LotteryActivity holds the schema definition for the LotteryActivity entity.
//
// 抽奖活动主表
type LotteryActivity struct {
	ent.Schema
}

func (LotteryActivity) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_activities"},
	}
}

func (LotteryActivity) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			MaxLen(200).
			NotEmpty().
			Comment("活动标题"),
		field.String("description").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Comment("活动描述"),
		field.String("share_code").
			MaxLen(32).
			NotEmpty().
			Unique().
			Comment("分享短码"),
		field.String("status").
			MaxLen(20).
			Default("pending").
			Comment("状态: pending/active/drawing/completed/cancelled"),
		field.Time("draw_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("开奖时间"),
		field.Time("activity_start_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("活动开始时间"),
		field.Time("activity_end_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("活动结束时间"),
		field.Int("min_participants").
			Default(10).
			Comment("最低参与人数"),
		field.Float("base_win_rate").
			SchemaType(map[string]string{dialect.Postgres: "decimal(5,4)"}).
			Default(0.1).
			Comment("基础中奖率"),
		field.String("weight_config").
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Default(`{"new_user":3.0,"regular":1.0,"paid":0.3,"subscriber":0.1}`).
			Comment("权重配置 JSON"),
		field.Int("winner_discount_percent").
			Default(95).
			Comment("中奖折扣百分比"),
		field.Float("loser_coupon_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Default(30.00).
			Comment("未中奖立减金额"),
		field.Int64("temp_group_id").
			Optional().
			Nillable().
			Comment("临时分组 ID"),
		field.Int("participant_count").
			Default(0).
			Comment("参与人数"),
		field.Int("winner_count").
			Default(0).
			Comment("中奖人数"),
		field.Int64("created_by").
			Comment("创建管理员 ID"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (LotteryActivity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("participants", LotteryParticipant.Type),
		edge.To("coupons", LotteryCoupon.Type),
	}
}

func (LotteryActivity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("draw_at"),
		index.Fields("activity_end_at"),
	}
}
