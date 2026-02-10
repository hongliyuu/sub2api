package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// BalanceLot holds the schema definition for the BalanceLot entity.
// 余额批次：每次资金入账创建一条记录，消费时按过期时间顺序扣减（FEFO）
type BalanceLot struct {
	ent.Schema
}

func (BalanceLot) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "balance_lots"},
	}
}

func (BalanceLot) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (BalanceLot) Fields() []ent.Field {
	return []ent.Field{
		// 用户ID
		field.Int64("user_id").
			Positive(),

		// 来源类型：recharge（充值）, redeem（兑换码）, promo（优惠码）, adjust（管理员调整）, migration（迁移）
		field.String("source_type").
			MaxLen(20).
			NotEmpty(),

		// 关联来源引用（订单号、兑换码等）
		field.String("source_ref").
			MaxLen(100).
			Optional().
			Nillable(),

		// 入账原始金额
		field.Float("original_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),

		// 当前剩余金额
		field.Float("remaining_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),

		// 批次状态：active（活跃）, depleted（已耗尽）, expired（已过期）
		field.String("status").
			MaxLen(20).
			Default("active"),

		// 过期时间
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),

		// 实际过期处理时间（过期调度器处理时设置）
		field.Time("expired_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Optional().
			Nillable(),

		// 描述
		field.String("description").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
	}
}

func (BalanceLot) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("balance_lots").
			Field("user_id").
			Required().
			Unique(),
	}
}

func (BalanceLot) Indexes() []ent.Index {
	return []ent.Index{
		// 扣费查询：按用户查活跃批次，按过期时间排序
		index.Fields("user_id", "status", "expires_at"),
		// 过期调度器查询：查找已过期的活跃批次
		index.Fields("status", "expires_at"),
		// 用户ID索引
		index.Fields("user_id"),
	}
}
