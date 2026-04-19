package schema

import (
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

var pendingAuthIntents = map[string]struct{}{
	"login":                        {},
	"bind_current_user":            {},
	"adopt_existing_user_by_email": {},
}

func validatePendingAuthIntent(value string) error {
	if _, ok := pendingAuthIntents[value]; ok {
		return nil
	}
	return fmt.Errorf("invalid pending auth intent %q", value)
}

// PendingAuthSession stores a short-lived post-OAuth decision session.
type PendingAuthSession struct {
	ent.Schema
}

func (PendingAuthSession) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "pending_auth_sessions"},
	}
}

func (PendingAuthSession) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (PendingAuthSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("intent").
			MaxLen(40).
			NotEmpty().
			Validate(validatePendingAuthIntent),
		field.String("provider_type").
			MaxLen(20).
			NotEmpty().
			Validate(validateAuthProviderType),
		field.String("provider_key").
			NotEmpty().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.String("provider_subject").
			NotEmpty().
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.JSON("upstream_identity_payload", map[string]any{}).
			Default(func() map[string]any { return map[string]any{} }).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int64("target_user_id").
			Optional().
			Nillable(),
		field.String("state_hash").
			MaxLen(255).
			NotEmpty(),
		field.String("nonce").
			MaxLen(255).
			NotEmpty(),
		field.String("redirect_to").
			Default("").
			SchemaType(map[string]string{dialect.Postgres: "text"}),
		field.Time("expires_at").
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("consumed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (PendingAuthSession) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("target_user", User.Type).
			Ref("pending_auth_sessions").
			Field("target_user_id").
			Unique(),
	}
}

func (PendingAuthSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("state_hash").Unique(),
		index.Fields("nonce").Unique(),
		index.Fields("target_user_id"),
		index.Fields("expires_at"),
		index.Fields("provider_type", "provider_key", "provider_subject"),
	}
}
