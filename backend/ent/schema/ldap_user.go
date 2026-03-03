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
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// LdapUser holds the schema definition for the LdapUser entity.
type LdapUser struct {
	ent.Schema
}

// Annotations of the LdapUser.
func (LdapUser) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ldap_users"},
	}
}

// Mixin of the LdapUser.
func (LdapUser) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

// Fields of the LdapUser.
func (LdapUser) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Unique().
			Comment("Foreign key to users table"),

		field.String("ldap_username").
			MaxLen(255).
			NotEmpty().
			Unique().
			Comment("LDAP username (uid)"),

		field.String("ldap_dn").
			MaxLen(500).
			NotEmpty().
			Comment("LDAP Distinguished Name"),

		field.Time("last_sync_at").
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}).
			Default(time.Now).
			Comment("Last synchronization time"),
	}
}

// Edges of the LdapUser.
func (LdapUser) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("user", User.Type).
			Required().
			Unique().
			Field("user_id").
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

// Indexes of the LdapUser.
func (LdapUser) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("ldap_username"),
	}
}
