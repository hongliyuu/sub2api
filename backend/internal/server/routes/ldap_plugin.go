// ldap_plugin.go 提供 LDAP 登录路由的懒绑定机制。
// 路由在服务器初始化时注册（指向此处的代理函数），
// LDAP 插件在 Setup() 中调用 RegisterLdapLoginHandler 注册实际处理器。
// 未注册处理器时，路由返回 404，从而实现插件可选加载。
package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ldapLoginHandler 运行时注册的 LDAP 登录处理器，未注册时为 nil
var ldapLoginHandler gin.HandlerFunc

// RegisterLdapLoginHandler 供 LDAP 插件在 Setup() 中注册登录处理器
func RegisterLdapLoginHandler(h gin.HandlerFunc) {
	ldapLoginHandler = h
}

// getLdapLoginHandler 返回已注册的 LDAP 登录处理器，
// 若未注册则返回一个 404 响应处理器（LDAP 未启用）
func getLdapLoginHandler() gin.HandlerFunc {
	if ldapLoginHandler == nil {
		return func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "LDAP_NOT_ENABLED",
				"message": "LDAP 认证未启用",
			})
		}
	}
	return ldapLoginHandler
}
