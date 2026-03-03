//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/suite"
)

type LdapUserRepoSuite struct {
	suite.Suite
	ctx      context.Context
	client   *dbent.Client
	repo     *ldapUserRepository
	userRepo *userRepository
}

func (s *LdapUserRepoSuite) SetupTest() {
	s.ctx = context.Background()
	s.client = testEntClient(s.T())
	s.repo = &ldapUserRepository{client: s.client}
	s.userRepo = newUserRepositoryWithSQL(s.client, integrationDB)

	// 清理测试数据
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM ldap_users")
	_, _ = integrationDB.ExecContext(s.ctx, "DELETE FROM users")
}

func TestLdapUserRepoSuite(t *testing.T) {
	suite.Run(t, new(LdapUserRepoSuite))
}

// mustCreateUser 创建测试用户
func (s *LdapUserRepoSuite) mustCreateUser(email string) *service.User {
	s.T().Helper()

	u := &service.User{
		Email:        email,
		PasswordHash: "test-password-hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	}

	s.Require().NoError(s.userRepo.Create(s.ctx, u), "create user")
	return u
}

// mustCreateLdapUser 创建测试 LDAP 用户
func (s *LdapUserRepoSuite) mustCreateLdapUser(userID int64, username, dn string) *service.LdapUser {
	s.T().Helper()

	ldapUser := &service.LdapUser{
		UserID:       userID,
		LdapUsername: username,
		LdapDn:       dn,
		LastSyncAt:   time.Now(),
	}

	s.Require().NoError(s.repo.Create(s.ctx, ldapUser), "create ldap user")
	return ldapUser
}

func (s *LdapUserRepoSuite) TestGetByEmailWithUser() {
	// 创建测试用户
	email := "test-ldap@example.com"
	user := s.mustCreateUser(email)

	// 创建 LDAP 用户关联
	username := "testuser"
	dn := "cn=testuser,ou=users,dc=example,dc=com"
	ldapUser := s.mustCreateLdapUser(user.ID, username, dn)

	// 测试通过邮箱查询
	result, err := s.repo.GetByEmailWithUser(s.ctx, email)
	s.Require().NoError(err, "GetByEmailWithUser should succeed")
	s.Require().NotNil(result, "result should not be nil")

	// 验证 LDAP 用户信息
	s.Equal(ldapUser.ID, result.ID)
	s.Equal(ldapUser.UserID, result.UserID)
	s.Equal(ldapUser.LdapUsername, result.LdapUsername)
	s.Equal(ldapUser.LdapDn, result.LdapDn)

	// 验证关联的用户信息
	s.Require().NotNil(result.User, "user should be loaded")
	s.Equal(user.ID, result.User.ID)
	s.Equal(user.Email, result.User.Email)
}

func (s *LdapUserRepoSuite) TestGetByEmailWithUser_NotFound() {
	// 测试不存在的邮箱
	result, err := s.repo.GetByEmailWithUser(s.ctx, "nonexistent@example.com")
	s.Error(err, "should return error for nonexistent email")
	s.Nil(result, "result should be nil")
	s.ErrorIs(err, service.ErrLdapUserNotFound, "should return ErrLdapUserNotFound")
}

func (s *LdapUserRepoSuite) TestUpdateUsernameAndDN() {
	// 创建测试用户和 LDAP 用户
	user := s.mustCreateUser("test-update@example.com")
	ldapUser := s.mustCreateLdapUser(user.ID, "olduser", "cn=olduser,ou=users,dc=example,dc=com")

	// 更新用户名和 DN
	newUsername := "newuser"
	newDN := "cn=newuser,ou=users,dc=example,dc=com"
	err := s.repo.UpdateUsernameAndDN(s.ctx, ldapUser.ID, newUsername, newDN)
	s.Require().NoError(err, "UpdateUsernameAndDN should succeed")

	// 验证更新结果
	updated, err := s.repo.GetByUsername(s.ctx, newUsername)
	s.Require().NoError(err, "should find updated user")
	s.Equal(ldapUser.ID, updated.ID)
	s.Equal(newUsername, updated.LdapUsername)
	s.Equal(newDN, updated.LdapDn)

	// 验证旧用户名不存在
	exists, err := s.repo.ExistsByUsername(s.ctx, "olduser")
	s.Require().NoError(err)
	s.False(exists, "old username should not exist")
}
