//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ldapUserRepoStub 用于测试的 LDAP 用户仓库 stub
type ldapUserRepoStub struct {
	getByUsernameFunc      func(ctx context.Context, username string) (*LdapUser, error)
	getByEmailWithUserFunc func(ctx context.Context, email string) (*LdapUser, error)
	updateUsernameAndDNFunc func(ctx context.Context, id int64, username, dn string) error
	createFunc             func(ctx context.Context, ldapUser *LdapUser) error
}

func (s *ldapUserRepoStub) Create(ctx context.Context, ldapUser *LdapUser) error {
	if s.createFunc != nil {
		return s.createFunc(ctx, ldapUser)
	}
	panic("unexpected Create call")
}

func (s *ldapUserRepoStub) GetByUsername(ctx context.Context, username string) (*LdapUser, error) {
	if s.getByUsernameFunc != nil {
		return s.getByUsernameFunc(ctx, username)
	}
	panic("unexpected GetByUsername call")
}

func (s *ldapUserRepoStub) GetByUserID(ctx context.Context, userID int64) (*LdapUser, error) {
	panic("unexpected GetByUserID call")
}

func (s *ldapUserRepoStub) UpdateLastSync(ctx context.Context, id int64) error {
	panic("unexpected UpdateLastSync call")
}

func (s *ldapUserRepoStub) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	panic("unexpected ExistsByUsername call")
}

func (s *ldapUserRepoStub) GetByEmailWithUser(ctx context.Context, email string) (*LdapUser, error) {
	if s.getByEmailWithUserFunc != nil {
		return s.getByEmailWithUserFunc(ctx, email)
	}
	panic("unexpected GetByEmailWithUser call")
}

func (s *ldapUserRepoStub) UpdateUsernameAndDN(ctx context.Context, id int64, username, dn string) error {
	if s.updateUsernameAndDNFunc != nil {
		return s.updateUsernameAndDNFunc(ctx, id, username, dn)
	}
	panic("unexpected UpdateUsernameAndDN call")
}

// userRepoStubWithUpdate 扩展 userRepoStub 以支持 Update 方法
type userRepoStubWithUpdate struct {
	*userRepoStub
	updateFunc func(ctx context.Context, user *User) error
}

func (s *userRepoStubWithUpdate) Update(ctx context.Context, user *User) error {
	if s.updateFunc != nil {
		return s.updateFunc(ctx, user)
	}
	panic("unexpected Update call")
}


// TestLdapErrorDefinitions 测试 LDAP 错误定义
func TestLdapErrorDefinitions(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedCode   string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "ErrLdapMultipleUsersFound",
			err:            ErrLdapMultipleUsersFound,
			expectedCode:   "LDAP_MULTIPLE_USERS_FOUND",
			expectedStatus: 400,
			expectedMsg:    "LDAP search returned multiple users. Please contact administrator to fix LDAP filter configuration.",
		},
		{
			name:           "ErrLdapUserEmailRequired",
			err:            ErrLdapUserEmailRequired,
			expectedCode:   "LDAP_USER_EMAIL_REQUIRED",
			expectedStatus: 400,
			expectedMsg:    "LDAP user must have email attribute (mail or userPrincipalName). Please contact administrator.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证错误类型
			appErr, ok := tt.err.(*infraerrors.ApplicationError)
			assert.True(t, ok, "错误应该是 ApplicationError 类型")

			// 验证错误原因码
			assert.Equal(t, tt.expectedCode, appErr.Reason, "错误原因码应该匹配")

			// 验证 HTTP 状态码
			assert.Equal(t, int32(tt.expectedStatus), appErr.Code, "HTTP 状态码应该是 400")

			// 验证错误消息
			assert.Equal(t, tt.expectedMsg, appErr.Message, "错误消息应该匹配")
		})
	}
}

// TestWithRetry 测试重试逻辑
func TestWithRetry(t *testing.T) {
	ctx := context.Background()

	t.Run("成功无需重试", func(t *testing.T) {
		callCount := 0
		fn := func(ctx context.Context) (*User, error) {
			callCount++
			return &User{ID: 1, Email: "test@example.com"}, nil
		}

		user, err := withRetry(ctx, 3, fn)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, 1, callCount, "应该只调用一次")
	})

	t.Run("并发冲突后重试成功", func(t *testing.T) {
		callCount := 0
		fn := func(ctx context.Context) (*User, error) {
			callCount++
			if callCount < 3 {
				return nil, ErrConcurrentConflict
			}
			return &User{ID: 2, Email: "test2@example.com"}, nil
		}

		user, err := withRetry(ctx, 3, fn)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, int64(2), user.ID)
		assert.Equal(t, 3, callCount, "应该重试2次后成功")
	})

	t.Run("达到最大重试次数", func(t *testing.T) {
		callCount := 0
		fn := func(ctx context.Context) (*User, error) {
			callCount++
			return nil, ErrConcurrentConflict
		}

		user, err := withRetry(ctx, 3, fn)
		require.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, ErrConcurrentConflict)
		assert.Equal(t, 4, callCount, "应该调用4次（初始1次+重试3次）")
	})

	t.Run("非并发冲突错误不重试", func(t *testing.T) {
		callCount := 0
		otherErr := errors.New("其他错误")
		fn := func(ctx context.Context) (*User, error) {
			callCount++
			return nil, otherErr
		}

		user, err := withRetry(ctx, 3, fn)
		require.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, otherErr)
		assert.Equal(t, 1, callCount, "不应该重试")
	})
}

// TestFindOrCreateUser_UsernameChange 测试用户名变更场景
func TestFindOrCreateUser_UsernameChange(t *testing.T) {
	ctx := context.Background()

	t.Run("检测到用户名变更并更新", func(t *testing.T) {
		// 准备测试数据
		oldUsername := "olduser"
		newUsername := "newuser"
		userEmail := "user@example.com"
		userDN := "cn=newuser,ou=users,dc=example,dc=com"

		existingUser := &User{
			ID:    1,
			Email: userEmail,
		}

		existingLdapUser := &LdapUser{
			ID:           10,
			UserID:       1,
			LdapUsername: oldUsername,
			LdapDn:       "cn=olduser,ou=users,dc=example,dc=com",
			User:         existingUser,
		}

		// Mock repositories
		ldapUserRepo := &ldapUserRepoStub{
			getByUsernameFunc: func(ctx context.Context, username string) (*LdapUser, error) {
				// 第一次查询新用户名，找不到
				if username == newUsername {
					return nil, ErrLdapUserNotFound
				}
				panic("unexpected username: " + username)
			},
			getByEmailWithUserFunc: func(ctx context.Context, email string) (*LdapUser, error) {
				// 通过邮箱找到旧的 LDAP 关联
				if email == userEmail {
					return existingLdapUser, nil
				}
				panic("unexpected email: " + email)
			},
			updateUsernameAndDNFunc: func(ctx context.Context, id int64, username, dn string) error {
				// 验证更新参数
				assert.Equal(t, existingLdapUser.ID, id)
				assert.Equal(t, newUsername, username)
				assert.Equal(t, userDN, dn)
				return nil
			},
		}

		// 创建支持 Update 的 userRepo stub
		userRepo := &userRepoStubWithUpdate{
			userRepoStub: &userRepoStub{},
			updateFunc: func(ctx context.Context, user *User) error {
				// 验证更新的用户
				assert.Equal(t, existingUser.ID, user.ID)
				assert.Equal(t, newUsername, user.Username)
				return nil
			},
		}

		// 创建 service
		service := &LdapService{
			ldapUserRepo: ldapUserRepo,
			userRepo:     userRepo,
		}

		// 执行测试
		user, err := service.findOrCreateUser(ctx, newUsername, userDN, userEmail)

		// 验证结果
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, existingUser.ID, user.ID)
		assert.Equal(t, existingUser.Email, user.Email)
	})
}

// TestFindOrCreateUser_ConcurrentConflict 测试并发冲突和重试
func TestFindOrCreateUser_ConcurrentConflict(t *testing.T) {
	ctx := context.Background()

	t.Run("创建LDAP关联时遇到并发冲突后重试成功", func(t *testing.T) {
		// 准备测试数据
		username := "testuser"
		userEmail := "test@example.com"
		userDN := "cn=testuser,ou=users,dc=example,dc=com"

		existingUser := &User{
			ID:    1,
			Email: userEmail,
		}

		createCallCount := 0

		// Mock repositories
		ldapUserRepo := &ldapUserRepoStub{
			getByUsernameFunc: func(ctx context.Context, username string) (*LdapUser, error) {
				// 第一次查询找不到
				return nil, ErrLdapUserNotFound
			},
			getByEmailWithUserFunc: func(ctx context.Context, email string) (*LdapUser, error) {
				// 通过邮箱也找不到
				return nil, ErrLdapUserNotFound
			},
			createFunc: func(ctx context.Context, ldapUser *LdapUser) error {
				createCallCount++
				if createCallCount == 1 {
					// 第一次创建遇到唯一约束冲突
					return errors.New("duplicate key value violates unique constraint")
				}
				// 第二次创建成功
				ldapUser.ID = 10
				return nil
			},
		}

		// 创建本地的 userRepo stub
		userRepo := &userRepoStub{
			user: existingUser,
		}

		// 创建 service
		service := &LdapService{
			ldapUserRepo: ldapUserRepo,
			userRepo:     userRepo,
		}

		// 执行测试
		user, err := service.findOrCreateUser(ctx, username, userDN, userEmail)

		// 验证结果
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, existingUser.ID, user.ID)
		assert.Equal(t, 2, createCallCount, "应该重试一次后成功")
	})
}
