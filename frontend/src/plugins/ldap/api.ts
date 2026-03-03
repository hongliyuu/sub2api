/**
 * LDAP 认证 API — 独立于主线 authAPI，避免污染主线模块。
 * 前端 API 路径 /auth/ldap/login 不变。
 */

import { apiClient } from '@/api/client'
import type { LoginResponse } from '@/api/auth'

/** LDAP 登录请求 */
export interface LdapLoginRequest {
  username: string
  password: string
  turnstile_token?: string
}

/**
 * LDAP 登录
 * @param credentials - LDAP 用户名和密码
 * @returns 登录响应（可能是完整认证或需要 2FA）
 */
export async function loginLdap(credentials: LdapLoginRequest): Promise<LoginResponse> {
  const { data } = await apiClient.post<LoginResponse>('/auth/ldap/login', credentials)
  return data
}
