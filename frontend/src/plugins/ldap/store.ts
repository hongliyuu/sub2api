/**
 * LDAP 认证 composable — 替代 useAuthStore 中已移除的 loginLdap action。
 * 负责调用 LDAP 登录 API 并将认证状态写入全局 authStore。
 */

import { useAuthStore } from '@/stores/auth'
import {
  isTotp2FARequired,
  type LoginResponse,
  setAuthToken,
  setRefreshToken,
  setTokenExpiresAt
} from '@/api/auth'
import { loginLdap } from './api'

const AUTH_USER_KEY = 'auth_user'

export function useLdapAuth() {
  const authStore = useAuthStore()

  /**
   * 执行 LDAP 登录并更新全局认证状态
   * @param username - LDAP 用户名
   * @param password - LDAP 密码
   * @param turnstileToken - 可选的 Turnstile 验证令牌
   */
  async function loginWithLdap(
    username: string,
    password: string,
    turnstileToken?: string
  ): Promise<LoginResponse> {
    const response = await loginLdap({ username, password, turnstile_token: turnstileToken })

    // 如果需要 2FA，直接返回，由视图层处理
    if (isTotp2FARequired(response)) {
      return response
    }

    // 先将 token 存入 localStorage（setToken 会从 localStorage 读取 refresh token）
    setAuthToken(response.access_token)
    if (response.refresh_token) {
      setRefreshToken(response.refresh_token)
    }
    if (response.expires_in) {
      setTokenExpiresAt(response.expires_in)
    }
    localStorage.setItem(AUTH_USER_KEY, JSON.stringify(response.user))

    // 通过 authStore.setToken 完成状态初始化（含 refreshUser + auto-refresh）
    await authStore.setToken(response.access_token)

    return response
  }

  return { loginWithLdap }
}
