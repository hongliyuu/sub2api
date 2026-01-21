// Package ldap provides LDAP authentication client.
package ldap

import (
	"crypto/tls"
	"fmt"

	"github.com/Wei-Shaw/sub2api/backend/internal/config"
	"github.com/go-ldap/ldap/v3"
)

// Client LDAP 客户端
type Client struct {
	config *config.LDAPConfig
	conn   *ldap.Conn
}

// UserInfo LDAP 用户信息
type UserInfo struct {
	Username    string // 用户名
	Email       string // 邮箱
	DisplayName string // 显示名称
	DN          string // 用户的完整 DN
}

// NewClient 创建 LDAP 客户端
func NewClient(cfg *config.LDAPConfig) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("ldap config is nil")
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("ldap is not enabled")
	}
	return &Client{config: cfg}, nil
}

// Connect 建立 LDAP 连接
func (c *Client) Connect() error {
	var conn *ldap.Conn
	var err error

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// 根据配置选择连接方式
	if c.config.UseTLS {
		// 使用 LDAPS（LDAP over SSL/TLS）
		tlsConfig := &tls.Config{
			InsecureSkipVerify: c.config.SkipTLSVerify,
			ServerName:         c.config.Host,
		}
		conn, err = ldap.DialTLS("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("ldap dial tls failed: %w", err)
		}
	} else {
		// 使用普通 LDAP
		conn, err = ldap.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("ldap dial failed: %w", err)
		}

		// 如果配置了 StartTLS，升级连接
		if c.config.UseStartTLS {
			tlsConfig := &tls.Config{
				InsecureSkipVerify: c.config.SkipTLSVerify,
				ServerName:         c.config.Host,
			}
			err = conn.StartTLS(tlsConfig)
			if err != nil {
				conn.Close()
				return fmt.Errorf("ldap start tls failed: %w", err)
			}
		}
	}

	c.conn = conn
	return nil
}

// Close 关闭 LDAP 连接
func (c *Client) Close() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	return nil
}

// Authenticate 认证用户
// 返回用户信息，如果认证失败返回错误
func (c *Client) Authenticate(username, password string) (*UserInfo, error) {
	// 建立连接
	if err := c.Connect(); err != nil {
		return nil, err
	}
	defer c.Close()

	// 使用管理员账号绑定，用于搜索用户
	err := c.conn.Bind(c.config.BindDN, c.config.BindPassword)
	if err != nil {
		return nil, fmt.Errorf("admin bind failed: %w", err)
	}

	// 搜索用户
	userInfo, err := c.SearchUser(username)
	if err != nil {
		return nil, err
	}

	// 使用用户凭证进行绑定验证
	err = c.conn.Bind(userInfo.DN, password)
	if err != nil {
		return nil, fmt.Errorf("user authentication failed: %w", err)
	}

	return userInfo, nil
}

// SearchUser 搜索用户信息
func (c *Client) SearchUser(username string) (*UserInfo, error) {
	// 构建搜索过滤器
	filter := fmt.Sprintf(c.config.UserFilter, ldap.EscapeFilter(username))

	// 构建搜索请求
	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, // 不限制结果数量
		0, // 不限制搜索时间
		false,
		filter,
		[]string{
			c.config.Attributes.Username,
			c.config.Attributes.Email,
			c.config.Attributes.DisplayName,
		},
		nil,
	)

	// 执行搜索
	result, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("ldap search failed: %w", err)
	}

	// 检查结果
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	if len(result.Entries) > 1 {
		return nil, fmt.Errorf("multiple users found: %s", username)
	}

	entry := result.Entries[0]

	// 提取用户信息
	userInfo := &UserInfo{
		DN:          entry.DN,
		Username:    entry.GetAttributeValue(c.config.Attributes.Username),
		Email:       entry.GetAttributeValue(c.config.Attributes.Email),
		DisplayName: entry.GetAttributeValue(c.config.Attributes.DisplayName),
	}

	// 如果用户名为空，使用搜索的用户名
	if userInfo.Username == "" {
		userInfo.Username = username
	}

	return userInfo, nil
}
