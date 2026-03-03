// Package ldap 提供 LDAP 认证功能的插件实现。
// 通过 init() 注册到全局插件系统，在 Wire 完成依赖注入后由 plugin.SetupAll() 调用 Setup()。
// 删除 cmd/server/plugins.go 中的空白导入即可完全禁用此插件，主线代码无需任何改动。
package ldap

import (
	"context"
	"log"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/plugin"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/server/routes"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func init() {
	// 在 main() 执行前通过 init() 注册插件
	plugin.Register(&ldapPlugin{})
}

// ldapPlugin 实现 plugin.Plugin 接口
type ldapPlugin struct{}

// Setup 在 Wire 依赖注入完成后由主程序调用，初始化 LDAP 相关组件并注册到主线系统
func (p *ldapPlugin) Setup(ctx *plugin.AppContext) {
	// 构建 LDAP 专属仓库（直接使用 EntClient，不进入 Wire 图）
	ldapConfigRepo := repository.NewLdapConfigRepository(ctx.EntClient)
	ldapUserRepo := repository.NewLdapUserRepository(ctx.EntClient)

	// 构建 LDAP 服务
	ldapSvc := service.NewLdapService(ldapConfigRepo, ldapUserRepo, ctx.UserRepo, ctx.Config)

	// 从环境变量加载 LDAP 配置（替代原来在 provideCleanup 中的逻辑）
	bgCtx := context.Background()
	if err := ldapSvc.LoadConfigFromEnv(bgCtx); err != nil {
		log.Printf("[LDAP] 从环境变量加载配置失败: %v", err)
	} else {
		log.Println("[LDAP] 已从环境变量加载配置")
	}

	// 构建 LDAP 认证 Handler 并注册路由处理器
	ldapHandler := handler.NewLdapAuthHandler(ldapSvc, ctx.AuthService, ctx.TotpService, ctx.SettingService)
	routes.RegisterLdapLoginHandler(ldapHandler.Login)

	log.Println("[LDAP] 插件初始化完成")
}
