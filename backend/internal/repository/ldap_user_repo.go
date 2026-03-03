package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/ldapuser"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type ldapUserRepository struct {
	client *ent.Client
}

// NewLdapUserRepository creates a new LDAP user repository.
func NewLdapUserRepository(client *ent.Client) service.LdapUserRepository {
	return &ldapUserRepository{client: client}
}

func (r *ldapUserRepository) Create(ctx context.Context, ldapUser *service.LdapUser) error {
	if ldapUser == nil {
		return fmt.Errorf("ldapUser cannot be nil")
	}
	client := clientFromContext(ctx, r.client)
	created, err := client.LdapUser.
		Create().
		SetUserID(ldapUser.UserID).
		SetLdapUsername(ldapUser.LdapUsername).
		SetLdapDn(ldapUser.LdapDn).
		SetLastSyncAt(ldapUser.LastSyncAt).
		Save(ctx)
	if err != nil {
		return err
	}
	ldapUser.ID = created.ID
	ldapUser.CreatedAt = created.CreatedAt
	ldapUser.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *ldapUserRepository) GetByUsername(ctx context.Context, username string) (*service.LdapUser, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.LdapUser.
		Query().
		Where(ldapuser.LdapUsernameEQ(username)).
		WithUser().
		First(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLdapUserNotFound, nil)
	}
	return ldapUserEntityToService(m), nil
}

func (r *ldapUserRepository) GetByUserID(ctx context.Context, userID int64) (*service.LdapUser, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.LdapUser.
		Query().
		Where(ldapuser.UserIDEQ(userID)).
		First(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLdapUserNotFound, nil)
	}
	return ldapUserEntityToService(m), nil
}

func (r *ldapUserRepository) UpdateLastSync(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	return client.LdapUser.
		UpdateOneID(id).
		SetLastSyncAt(time.Now()).
		Exec(ctx)
}

func (r *ldapUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	client := clientFromContext(ctx, r.client)
	return client.LdapUser.
		Query().
		Where(ldapuser.LdapUsernameEQ(username)).
		Exist(ctx)
}

// GetByEmailWithUser 通过邮箱查询 LDAP 用户关联（包含关联的用户信息）
func (r *ldapUserRepository) GetByEmailWithUser(ctx context.Context, email string) (*service.LdapUser, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.LdapUser.
		Query().
		Where(ldapuser.HasUserWith(user.EmailEQ(email))).
		WithUser().
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrLdapUserNotFound, nil)
	}
	return ldapUserEntityToService(m), nil
}

// UpdateUsernameAndDN 更新 LDAP 用户名和 DN
func (r *ldapUserRepository) UpdateUsernameAndDN(ctx context.Context, id int64, username, dn string) error {
	client := clientFromContext(ctx, r.client)
	err := client.LdapUser.
		UpdateOneID(id).
		SetLdapUsername(username).
		SetLdapDn(dn).
		SetLastSyncAt(time.Now()).
		Exec(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, nil)
	}
	return nil
}

func ldapUserEntityToService(m *ent.LdapUser) *service.LdapUser {
	if m == nil {
		return nil
	}
	out := &service.LdapUser{
		ID:           m.ID,
		UserID:       m.UserID,
		LdapUsername: m.LdapUsername,
		LdapDn:       m.LdapDn,
		LastSyncAt:   m.LastSyncAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	if m.Edges.User != nil {
		out.User = userEntityToService(m.Edges.User)
	}
	return out
}
