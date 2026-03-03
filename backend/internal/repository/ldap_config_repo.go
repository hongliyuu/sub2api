package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/ldapconfig"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type ldapConfigRepository struct {
	client *ent.Client
}

// NewLdapConfigRepository creates a new LDAP config repository.
func NewLdapConfigRepository(client *ent.Client) service.LdapConfigRepository {
	return &ldapConfigRepository{client: client}
}

func (r *ldapConfigRepository) Get(ctx context.Context) (*service.LdapConfig, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.LdapConfig.
		Query().
		Order(ent.Asc(ldapconfig.FieldID)).
		First(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLdapConfigNotFound, nil)
	}
	return ldapConfigEntityToService(m), nil
}

func (r *ldapConfigRepository) GetEnabled(ctx context.Context) (*service.LdapConfig, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.LdapConfig.
		Query().
		Where(ldapconfig.EnabledEQ(true)).
		Order(ent.Asc(ldapconfig.FieldID)).
		First(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLdapConfigNotFound, nil)
	}
	return ldapConfigEntityToService(m), nil
}

func (r *ldapConfigRepository) Create(ctx context.Context, config *service.LdapConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	client := clientFromContext(ctx, r.client)
	created, err := client.LdapConfig.
		Create().
		SetServerURL(config.ServerURL).
		SetBindDn(config.BindDn).
		SetBindPasswordEncrypted(config.BindPasswordEncrypted).
		SetBaseDn(config.BaseDn).
		SetUserFilter(config.UserFilter).
		SetEnabled(config.Enabled).
		SetTLSEnabled(config.TLSEnabled).
		SetTLSSkipVerify(config.TLSSkipVerify).
		SetConfigSource(config.ConfigSource).
		Save(ctx)
	if err != nil {
		return err
	}
	config.ID = created.ID
	config.CreatedAt = created.CreatedAt
	config.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *ldapConfigRepository) Update(ctx context.Context, config *service.LdapConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}
	client := clientFromContext(ctx, r.client)
	updated, err := client.LdapConfig.
		UpdateOneID(config.ID).
		SetServerURL(config.ServerURL).
		SetBindDn(config.BindDn).
		SetBindPasswordEncrypted(config.BindPasswordEncrypted).
		SetBaseDn(config.BaseDn).
		SetUserFilter(config.UserFilter).
		SetEnabled(config.Enabled).
		SetTLSEnabled(config.TLSEnabled).
		SetTLSSkipVerify(config.TLSSkipVerify).
		SetConfigSource(config.ConfigSource).
		Save(ctx)
	if err != nil {
		return err
	}
	config.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *ldapConfigRepository) Exists(ctx context.Context) (bool, error) {
	client := clientFromContext(ctx, r.client)
	return client.LdapConfig.Query().Exist(ctx)
}

func ldapConfigEntityToService(m *ent.LdapConfig) *service.LdapConfig {
	if m == nil {
		return nil
	}
	return &service.LdapConfig{
		ID:                    m.ID,
		ServerURL:             m.ServerURL,
		BindDn:                m.BindDn,
		BindPasswordEncrypted: m.BindPasswordEncrypted,
		BaseDn:                m.BaseDn,
		UserFilter:            m.UserFilter,
		Enabled:               m.Enabled,
		TLSEnabled:            m.TLSEnabled,
		TLSSkipVerify:         m.TLSSkipVerify,
		ConfigSource:          m.ConfigSource,
		CreatedAt:             m.CreatedAt,
		UpdatedAt:             m.UpdatedAt,
	}
}
