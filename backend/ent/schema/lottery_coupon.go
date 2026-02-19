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

// LotteryCoupon holds the schema definition for the LotteryCoupon entity.
//
// 抽奖优惠券
type LotteryCoupon struct {
	ent.Schema
}

func (LotteryCoupon) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_coupons"},
	}
}

func (LotteryCoupon) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("activity_id").
			Comment("活动 ID"),
		field.Int64("user_id").
			Comment("用户 ID"),
		field.String("coupon_type").
			MaxLen(20).
			Comment("优惠券类型: winner_discount/loser_reduction"),
		field.Int("discount_percent").
			Optional().
			Nillable().
			Comment("折扣百分比（95=付95%）"),
		field.Float("reduction_amount").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,2)"}).
			Comment("立减金额"),
		field.String("applicable_scope").
			MaxLen(50).
			Comment("适用范围: recharge/monthly_subscription"),
		field.String("status").
			MaxLen(20).
			Default("active").
			Comment("状态: active/used/expired"),
		field.Time("used_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("使用时间"),
		field.String("used_order_id").
			Optional().
			Nillable().
			MaxLen(100).
			Comment("关联订单号"),
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("过期时间"),
		field.Bool("reminded").
			Default(false).
			Comment("过期前是否已提醒"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (LotteryCoupon) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("activity", LotteryActivity.Type).
			Ref("coupons").
			Field("activity_id").
			Unique().
			Required(),
	}
}

func (LotteryCoupon) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "status"),
		index.Fields("activity_id"),
		index.Fields("expires_at"),
	}
}
