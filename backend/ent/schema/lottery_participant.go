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

// LotteryParticipant holds the schema definition for the LotteryParticipant entity.
//
// 抽奖参与记录
type LotteryParticipant struct {
	ent.Schema
}

func (LotteryParticipant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "lottery_participants"},
	}
}

func (LotteryParticipant) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("activity_id").
			Comment("活动 ID"),
		field.Int64("user_id").
			Comment("用户 ID"),
		field.String("user_category").
			MaxLen(20).
			Comment("用户类别: new_user/regular/paid/subscriber"),
		field.Float("weight_multiplier").
			SchemaType(map[string]string{dialect.Postgres: "decimal(5,2)"}).
			Default(1.0).
			Comment("权重倍数"),
		field.Bool("is_winner").
			Optional().
			Nillable().
			Comment("是否中奖: NULL=未开奖"),
		field.Int64("coupon_id").
			Optional().
			Nillable().
			Comment("关联优惠券 ID"),
		field.Time("participated_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}).
			Comment("参与时间"),
	}
}

func (LotteryParticipant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("activity", LotteryActivity.Type).
			Ref("participants").
			Field("activity_id").
			Unique().
			Required(),
	}
}

func (LotteryParticipant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("activity_id", "user_id").Unique(),
		index.Fields("activity_id"),
		index.Fields("user_id"),
	}
}
