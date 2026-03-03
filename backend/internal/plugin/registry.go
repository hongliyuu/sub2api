// Package plugin 提供插件注册与初始化机制。
// 插件通过 init() 调用 Register() 注册自身，
// 主程序在 Wire 初始化完成后调用 SetupAll() 一次性初始化所有插件。
package plugin

import (
	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// AppContext 插件初始化所需的核心依赖上下文，由 Wire 注入并传递给每个插件的 Setup 方法
type AppContext struct {
	EntClient      *ent.Client
	Config         *config.Config
	SettingService *service.SettingService
	TotpService    *service.TotpService
	AuthService    *service.AuthService
	UserRepo       service.UserRepository
}

// Plugin 定义插件接口，所有插件必须实现此接口
type Plugin interface {
	Setup(ctx *AppContext)
}

// registeredPlugins 存储所有已注册的插件
var registeredPlugins []Plugin

// Register 注册一个插件，通常在插件包的 init() 函数中调用
func Register(p Plugin) {
	registeredPlugins = append(registeredPlugins, p)
}

// SetupAll 初始化所有已注册的插件，在服务器启动前调用
func SetupAll(ctx *AppContext) {
	for _, p := range registeredPlugins {
		p.Setup(ctx)
	}
}
