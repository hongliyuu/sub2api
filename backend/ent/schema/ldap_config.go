package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

// LdapConfig holds the schema definition for the LdapConfig entity.
type LdapConfig struct {
	ent.Schema
}

// Annotations of the LdapConfig.
func (LdapConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "ldap_configs"},
	}
}

// Mixin of the LdapConfig.
func (LdapConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

// Fields of the LdapConfig.
func (LdapConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("server_url").
			MaxLen(255).
			NotEmpty().
			Comment("LDAP server URL (e.g., ldap://ldap.example.com:389)"),

		field.String("bind_dn").
			MaxLen(255).
			NotEmpty().
			Comment("Bind DN for LDAP authentication"),

		field.String("bind_password_encrypted").
			SchemaType(map[string]string{
				dialect.Postgres: "text",
			}).
			NotEmpty().
			Sensitive().
			Comment("Encrypted bind password (AES-256-GCM)"),

		field.String("base_dn").
			MaxLen(255).
			NotEmpty().
			Comment("Base DN for user search"),

		field.String("user_filter").
			MaxLen(255).
			Default("(uid=%s)").
			Comment("LDAP user filter template"),

		field.Bool("enabled").
			Default(false).
			Comment("Whether this LDAP config is enabled"),

		field.Bool("tls_enabled").
			Default(false).
			Comment("Whether to use LDAP over TLS (ldaps://)"),

		field.Bool("tls_skip_verify").
			Default(false).
			Comment("Whether to skip TLS certificate verification"),

		field.String("config_source").
			MaxLen(50).
			Default("database").
			Comment("Configuration source: 'env' or 'database'"),
	}
}

// Indexes of the LdapConfig.
func (LdapConfig) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled"),
	}
}
