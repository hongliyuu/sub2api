// plugins.go 是 LDAP 插件的唯一注册点。
// 删除此文件（或移除空白导入）即可完全禁用 LDAP 功能，主线代码无需任何改动。
package main

import (
	// 引入 LDAP 插件，通过 init() 注册到全局插件系统
	_ "github.com/Wei-Shaw/sub2api/internal/plugins/ldap"
)
