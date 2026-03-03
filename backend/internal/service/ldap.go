package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/go-ldap/ldap/v3"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrLdapConfigNotFound       = infraerrors.InternalServer("LDAP_CONFIG_NOT_FOUND", "ldap configuration not found")
	ErrLdapUserNotFound         = infraerrors.Unauthorized("LDAP_USER_NOT_FOUND", "invalid ldap credentials")
	ErrLdapDisabled             = infraerrors.Forbidden("LDAP_DISABLED", "ldap authentication is disabled")
	ErrLdapConnectionFailed     = infraerrors.ServiceUnavailable("LDAP_CONNECTION_FAILED", "failed to connect to ldap server")
	ErrLdapBindFailed           = infraerrors.Unauthorized("LDAP_BIND_FAILED", "invalid ldap credentials")
	ErrLdapInvalidCredentials   = infraerrors.Unauthorized("LDAP_INVALID_CREDENTIALS", "invalid ldap credentials")
	ErrLdapEncryptionKeyNotSet  = infraerrors.InternalServer("LDAP_ENCRYPTION_KEY_NOT_SET", "ldap encryption key not configured")
	ErrLdapInvalidEncryptionKey = infraerrors.InternalServer("LDAP_INVALID_ENCRYPTION_KEY", "invalid ldap encryption key format")
	ErrLdapMultipleUsersFound   = infraerrors.BadRequest("LDAP_MULTIPLE_USERS_FOUND", "LDAP search returned multiple users. Please contact administrator to fix LDAP filter configuration.")
	ErrLdapUserEmailRequired    = infraerrors.BadRequest("LDAP_USER_EMAIL_REQUIRED", "LDAP user must have email attribute (mail or userPrincipalName). Please contact administrator.")
	ErrConcurrentConflict       = errors.New("concurrent conflict detected")
)

type LdapConfig struct {
	ID                    int64
	ServerURL             string
	BindDn                string
	BindPasswordEncrypted string
	BaseDn                string
	UserFilter            string
	Enabled               bool
	TLSEnabled            bool
	TLSSkipVerify         bool
	ConfigSource          string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type LdapUser struct {
	ID           int64
	UserID       int64
	LdapUsername string
	LdapDn       string
	LastSyncAt   time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	User         *User
}

type LdapConfigRepository interface {
	Get(ctx context.Context) (*LdapConfig, error)
	GetEnabled(ctx context.Context) (*LdapConfig, error)
	Create(ctx context.Context, config *LdapConfig) error
	Update(ctx context.Context, config *LdapConfig) error
	Exists(ctx context.Context) (bool, error)
}

type LdapUserRepository interface {
	Create(ctx context.Context, ldapUser *LdapUser) error
	GetByUsername(ctx context.Context, username string) (*LdapUser, error)
	GetByUserID(ctx context.Context, userID int64) (*LdapUser, error)
	UpdateLastSync(ctx context.Context, id int64) error
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	GetByEmailWithUser(ctx context.Context, email string) (*LdapUser, error)
	UpdateUsernameAndDN(ctx context.Context, id int64, username, dn string) error
}

// LdapService 处理 LDAP 认证操作
type LdapService struct {
	ldapConfigRepo LdapConfigRepository
	ldapUserRepo   LdapUserRepository
	userRepo       UserRepository
	cfg            *config.Config
}

// NewLdapService 创建新的 LDAP 服务实例
func NewLdapService(
	ldapConfigRepo LdapConfigRepository,
	ldapUserRepo LdapUserRepository,
	userRepo UserRepository,
	cfg *config.Config,
) *LdapService {
	return &LdapService{
		ldapConfigRepo: ldapConfigRepo,
		ldapUserRepo:   ldapUserRepo,
		userRepo:       userRepo,
		cfg:            cfg,
	}
}

// getEncryptionKeyBytes 获取并验证加密密钥
func getEncryptionKeyBytes() ([]byte, error) {
	key := os.Getenv("LDAP_ENCRYPTION_KEY")
	if key == "" {
		return nil, ErrLdapEncryptionKeyNotSet
	}

	keyBytes, err := hex.DecodeString(key)
	if err != nil || len(keyBytes) != 32 {
		return nil, ErrLdapInvalidEncryptionKey
	}

	return keyBytes, nil
}

// encryptPassword 使用 AES-256-GCM 加密密码
func (s *LdapService) encryptPassword(plaintext string) (string, error) {
	keyBytes, err := getEncryptionKeyBytes()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptPassword 使用 AES-256-GCM 解密密码
func (s *LdapService) decryptPassword(ciphertext string) (string, error) {
	keyBytes, err := getEncryptionKeyBytes()
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// connectLdap 建立 LDAP 连接（带 3 秒超时）
func (s *LdapService) connectLdap(config *LdapConfig) (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	// 创建带超时的 dialer（遵循规范要求的 3 秒超时）
	dialer := &net.Dialer{Timeout: 3 * time.Second}

	if config.TLSEnabled {
		// LDAPS 连接
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.TLSSkipVerify,
		}
		conn, err = ldap.DialURL(config.ServerURL,
			ldap.DialWithDialer(dialer),
			ldap.DialWithTLSConfig(tlsConfig))
	} else {
		// 普通 LDAP 连接
		conn, err = ldap.DialURL(config.ServerURL,
			ldap.DialWithDialer(dialer))
	}

	if err != nil {
		slog.Error("ldap_connection_failed",
			"server", maskServerURL(config.ServerURL),
			"error", err)
		return nil, ErrLdapConnectionFailed
	}

	slog.Debug("ldap_connection_established",
		"server", maskServerURL(config.ServerURL),
		"tls_enabled", config.TLSEnabled)

	return conn, nil
}

// searchUser 搜索 LDAP 用户并获取 DN 和邮箱
func (s *LdapService) searchUser(
	conn *ldap.Conn,
	config *LdapConfig,
	username, bindPassword string,
) (string, string, error) {
	// 1. 使用 bind DN 进行绑定
	if err := conn.Bind(config.BindDn, bindPassword); err != nil {
		slog.Error("ldap_bind_failed",
			"bind_dn", maskDN(config.BindDn),
			"error", err)
		return "", "", ErrLdapBindFailed
	}

	// 2. 构造搜索过滤器（转义用户输入防止 LDAP 注入）
	safeUsername := ldap.EscapeFilter(username)
	filter := fmt.Sprintf(config.UserFilter, safeUsername)

	// 3. 执行搜索
	searchRequest := ldap.NewSearchRequest(
		config.BaseDn,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		5, // 5 秒超时
		false,
		filter,
		[]string{"dn", "mail", "userPrincipalName"},
		nil,
	)

	result, err := conn.Search(searchRequest)
	if err != nil {
		slog.Error("ldap_search_failed",
			"filter", filter,
			"error", err)
		return "", "", ErrLdapUserNotFound
	}

	if len(result.Entries) == 0 {
		slog.Warn("ldap_user_not_found", "username", username)
		return "", "", ErrLdapUserNotFound
	}

	if len(result.Entries) > 1 {
		slog.Warn("ldap_multiple_users_found",
			"username", username,
			"count", len(result.Entries),
			"filter", filter)
		return "", "", ErrLdapMultipleUsersFound
	}

	entry := result.Entries[0]
	userDN := entry.DN
	userEmail := entry.GetAttributeValue("mail")
	if userEmail == "" {
		userEmail = entry.GetAttributeValue("userPrincipalName")
	}

	if userEmail == "" {
		slog.Warn("ldap_user_email_required",
			"username", username,
			"dn", maskDN(userDN))
		return "", "", ErrLdapUserEmailRequired
	}

	slog.Debug("ldap_user_found",
		"username", username,
		"dn", maskDN(userDN),
		"email", userEmail)

	return userDN, userEmail, nil
}

// bindUser 验证用户密码
func (s *LdapService) bindUser(conn *ldap.Conn, userDN, password string) error {
	if err := conn.Bind(userDN, password); err != nil {
		slog.Warn("ldap_user_bind_failed",
			"dn", maskDN(userDN),
			"error", err)
		return ErrLdapInvalidCredentials
	}

	slog.Info("ldap_authentication_success", "dn", maskDN(userDN))
	return nil
}

// maskServerURL 掩码服务器 URL 中的敏感信息
func maskServerURL(url string) string {
	if url == "" {
		return ""
	}
	// 保留协议和主机，隐藏端口和路径
	parts := strings.SplitN(url, "://", 2)
	if len(parts) != 2 {
		return "***"
	}
	protocol := parts[0]
	rest := parts[1]

	// 提取主机部分
	hostParts := strings.SplitN(rest, ":", 2)
	host := hostParts[0]

	// 只显示主机的前几个字符
	if len(host) > 10 {
		return fmt.Sprintf("%s://%s***", protocol, host[:10])
	}
	return fmt.Sprintf("%s://%s***", protocol, host)
}

// maskDN 掩码 DN 中的敏感信息
func maskDN(dn string) string {
	if dn == "" {
		return ""
	}
	// 只显示 DN 的第一个组件
	parts := strings.SplitN(dn, ",", 2)
	if len(parts) > 1 {
		return parts[0] + ",***"
	}
	return dn
}

// Authenticate 执行 LDAP 认证并返回本地用户
func (s *LdapService) Authenticate(ctx context.Context, username, password string) (*User, error) {
	// 1. 获取 LDAP 配置
	config, err := s.ldapConfigRepo.GetEnabled(ctx)
	if err != nil {
		if errors.Is(err, ErrLdapConfigNotFound) {
			return nil, ErrLdapDisabled
		}
		return nil, err
	}

	// 2. 解密 bind password
	bindPassword, err := s.decryptPassword(config.BindPasswordEncrypted)
	if err != nil {
		slog.Error("ldap_decrypt_password_failed", "error", err)
		return nil, ErrLdapConfigNotFound
	}

	// 3. 连接 LDAP 服务器
	conn, err := s.connectLdap(config)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			// 记录关闭错误，但不影响主流程
			_ = closeErr
		}
	}()

	// 4. 搜索用户 DN
	userDN, userEmail, err := s.searchUser(conn, config, username, bindPassword)
	if err != nil {
		return nil, err
	}

	// 5. 验证用户密码（Bind）
	if err := s.bindUser(conn, userDN, password); err != nil {
		return nil, err
	}

	// 6. 查找或创建本地用户
	user, err := s.findOrCreateUser(ctx, username, userDN, userEmail)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// findOrCreateUser 查找或创建本地用户（支持用户名变更和并发冲突重试）
func (s *LdapService) findOrCreateUser(
	ctx context.Context,
	username, userDN, userEmail string,
) (*User, error) {
	return withRetry(ctx, 3, func(ctx context.Context) (*User, error) {
		// 1. 尝试通过 LDAP username 查找
		ldapUser, err := s.ldapUserRepo.GetByUsername(ctx, username)
		if err == nil {
			// 找到关联，返回本地用户
			return ldapUser.User, nil
		}

		if !errors.Is(err, ErrLdapUserNotFound) {
			return nil, err
		}

		// 2. 尝试通过邮箱查找 LDAP 关联（用户名可能变更）
		ldapUser, err = s.ldapUserRepo.GetByEmailWithUser(ctx, userEmail)
		if err == nil {
			// 找到关联，检测用户名变更
			if ldapUser.LdapUsername != username {
				// 用户名变更，更新
				slog.Info("ldap_username_changed",
					"user_id", ldapUser.UserID,
					"old_username", ldapUser.LdapUsername,
					"new_username", username,
					"email", userEmail)

				// 更新用户名和 DN
				err = s.ldapUserRepo.UpdateUsernameAndDN(ctx, ldapUser.ID, username, userDN)
				if err != nil {
					// 检查是否是唯一约束冲突
					if ent.IsConstraintError(err) {
						return nil, ErrConcurrentConflict
					}
					return nil, err
				}

				// 同时更新本地用户的 username
				ldapUser.User.Username = username
				err = s.userRepo.Update(ctx, ldapUser.User)
				if err != nil {
					return nil, fmt.Errorf("failed to update user username: %w", err)
				}
			}
			return ldapUser.User, nil
		}

		if !errors.Is(err, ErrLdapUserNotFound) {
			return nil, err
		}

		// 3. 都找不到，尝试通过邮箱查找本地用户
		user, err := s.userRepo.GetByEmail(ctx, userEmail)
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			return nil, err
		}

		// 4. 如果用户不存在，创建新用户
		if user == nil {
			user, err = s.createLdapUser(ctx, username, userEmail)
			if err != nil {
				return nil, err
			}
		}

		// 5. 创建 LDAP 关联
		ldapUserEntity := &LdapUser{
			UserID:       user.ID,
			LdapUsername: username,
			LdapDn:       userDN,
			LastSyncAt:   time.Now(),
		}

		if err := s.ldapUserRepo.Create(ctx, ldapUserEntity); err != nil {
			// 检查是否是唯一约束冲突
			if ent.IsConstraintError(err) {
				return nil, ErrConcurrentConflict
			}

			slog.Error("ldap_user_association_failed",
				"user_id", user.ID,
				"username", username,
				"error", err)
			return nil, err
		}

		slog.Info("ldap_user_associated",
			"user_id", user.ID,
			"username", username)

		return user, nil
	})
}

// createLdapUser 创建 LDAP 用户（遵循 C14）
func (s *LdapService) createLdapUser(
	ctx context.Context,
	username, email string,
) (*User, error) {
	// 1. 生成随机密码（禁止本地密码登录）
	randomPassword, err := generateRandomHex(32)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := hashPassword(randomPassword)
	if err != nil {
		return nil, err
	}

	// 2. 获取默认配置
	defaultBalance := s.cfg.Default.UserBalance
	defaultConcurrency := s.cfg.Default.UserConcurrency

	// 3. 创建用户
	user := &User{
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         "user",
		Balance:      defaultBalance,
		Concurrency:  defaultConcurrency,
		Status:       "active",
		Username:     username,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		// 处理并发创建冲突（遵循 C14 约束）
		if errors.Is(err, ErrEmailExists) {
			// 重新加载已存在的用户
			existingUser, getErr := s.userRepo.GetByEmail(ctx, email)
			if getErr != nil {
				slog.Error("ldap_user_reload_failed",
					"username", username,
					"email", email,
					"error", getErr)
				return nil, getErr
			}
			slog.Info("ldap_user_already_exists",
				"user_id", existingUser.ID,
				"username", username,
				"email", email)
			return existingUser, nil
		}
		// 其他错误正常返回
		slog.Error("ldap_user_creation_failed",
			"username", username,
			"email", email,
			"error", err)
		return nil, err
	}

	slog.Info("ldap_user_created",
		"user_id", user.ID,
		"username", username,
		"email", email)

	return user, nil
}

// generateRandomHex 生成随机 hex 字符串
func generateRandomHex(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashPassword 使用 bcrypt 哈希密码
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// LoadConfigFromEnv 从环境变量加载配置
func (s *LdapService) LoadConfigFromEnv(ctx context.Context) error {
	// 检查环境变量是否存在
	serverURL := os.Getenv("LDAP_SERVER_URL")
	if serverURL == "" {
		return nil // 无环境变量配置
	}

	bindDN := os.Getenv("LDAP_BIND_DN")
	bindPassword := os.Getenv("LDAP_BIND_PASSWORD")
	baseDN := os.Getenv("LDAP_BASE_DN")

	if bindDN == "" || bindPassword == "" || baseDN == "" {
		return fmt.Errorf("incomplete LDAP environment variables")
	}

	// 加密密码
	encryptedPassword, err := s.encryptPassword(bindPassword)
	if err != nil {
		return err
	}

	// 构造配置对象
	config := &LdapConfig{
		ServerURL:             serverURL,
		BindDn:                bindDN,
		BindPasswordEncrypted: encryptedPassword,
		BaseDn:                baseDN,
		UserFilter:            getEnvOrDefault("LDAP_USER_FILTER", "(uid=%s)"),
		Enabled:               getEnvBool("LDAP_ENABLED", false),
		TLSEnabled:            getEnvBool("LDAP_TLS_ENABLED", false),
		TLSSkipVerify:         getEnvBool("LDAP_TLS_SKIP_VERIFY", false),
		ConfigSource:          "env",
	}

	// 检查是否已存在配置
	exists, err := s.ldapConfigRepo.Exists(ctx)
	if err != nil {
		return err
	}

	if exists {
		// 更新现有配置
		existingConfig, err := s.ldapConfigRepo.Get(ctx)
		if err != nil {
			return err
		}
		config.ID = existingConfig.ID
		return s.ldapConfigRepo.Update(ctx, config)
	}

	// 创建新配置
	return s.ldapConfigRepo.Create(ctx, config)
}

// getEnvOrDefault 获取环境变量或返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool 获取布尔类型的环境变量
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

// withRetry 重试辅助函数
func withRetry(ctx context.Context, maxRetries int, fn func(context.Context) (*User, error)) (*User, error) {
	retryDelays := []time.Duration{10 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		user, err := fn(ctx)
		if err == nil {
			return user, nil
		}

		// 只重试并发冲突错误
		if !errors.Is(err, ErrConcurrentConflict) {
			return nil, err
		}

		if attempt < maxRetries {
			slog.Debug("retrying_after_concurrent_conflict",
				"attempt", attempt+1,
				"delay_ms", retryDelays[attempt].Milliseconds())
			time.Sleep(retryDelays[attempt])
		}
	}

	return nil, ErrConcurrentConflict
}
