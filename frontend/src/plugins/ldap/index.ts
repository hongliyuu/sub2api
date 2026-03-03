/**
 * LDAP 前端插件入口。
 * 通过 app.use(ldapPlugin) 动态注册 /auth/ldap 路由，不污染主路由表。
 * 删除 main.ts 中的 app.use(ldapPlugin) 行即可完全禁用前端 LDAP 功能。
 */

import type { App } from 'vue'
import type { Router } from 'vue-router'

const ldapPlugin = {
  install(app: App) {
    // 从全局获取 router 实例（由 main.ts 在 app.use(router) 后注入）
    const router = app.config.globalProperties.$router as Router
    if (!router) {
      console.warn('[LDAP Plugin] Router not found, route /auth/ldap will not be registered')
      return
    }

    // 动态注册 LDAP 登录路由，不写死在主路由表
    router.addRoute({
      path: '/auth/ldap',
      name: 'LdapLogin',
      component: () => import('./views/LdapLoginView.vue'),
      meta: {
        requiresAuth: false,
        title: 'LDAP Login'
      }
    })
  }
}

export default ldapPlugin
