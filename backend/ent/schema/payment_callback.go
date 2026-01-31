package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PaymentCallback holds the schema definition for the PaymentCallback entity.
// 记录所有支付回调请求，用于审计和调试
type PaymentCallback struct {
	ent.Schema
}

func (PaymentCallback) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "payment_callbacks"},
	}
}

func (PaymentCallback) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (PaymentCallback) Fields() []ent.Field {
	return []ent.Field{
		// 订单号（从回调数据中解析）
		field.String("order_no").
			MaxLen(50).
			Optional().
			Nillable(),

		// 支付方式：wechat_pay, alipay
		field.String("payment_method").
			MaxLen(20).
			NotEmpty(),

		// 支付平台订单号（微信支付 transaction_id）
		field.String("transaction_id").
			MaxLen(64).
			Optional().
			Nillable(),

		// 请求头（JSON 格式）
		field.String("request_headers").
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}).
			Default("{}"),

		// 请求体（JSON 格式，加密前的原始数据）
		field.String("request_body").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),

		// 签名验证结果
		field.Bool("signature_valid").
			Default(false),

		// 处理状态：received（已接收）, processing（处理中）, success（处理成功）, failed（处理失败）
		field.String("process_status").
			MaxLen(20).
			Default("received"),

		// 处理结果描述
		field.String("process_message").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),

		// 响应码（返回给微信的响应）
		field.String("response_code").
			MaxLen(20).
			Default(""),

		// 响应消息
		field.String("response_message").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),

		// 处理耗时（毫秒）
		field.Int64("process_time_ms").
			Default(0),
	}
}

func (PaymentCallback) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_no"),
		index.Fields("transaction_id"),
		index.Fields("payment_method"),
		index.Fields("process_status"),
		index.Fields("created_at"),
	}
}
