/**
 * User API endpoints
 * Handles user profile management and password changes
 */

import { apiClient } from './client'
import type {
  User,
  ChangePasswordRequest,
  NotifyEmailEntry,
  UserAccountBindingProvider
} from '@/types'

type ProfileMutationResponse = User | { user?: User | null } | null

function extractUserFromMutationResponse(payload: ProfileMutationResponse): User | null {
  if (!payload) {
    return null
  }

  if ('id' in payload && 'email' in payload) {
    return payload
  }

  return payload.user ?? null
}

function isEndpointFallbackError(error: unknown): boolean {
  const status = (error as any)?.response?.status ?? (error as any)?.status
  return status === 404 || status === 405
}

async function requestFirstSupported<T>(requests: Array<() => Promise<T>>): Promise<T> {
  let fallbackError: unknown = null

  for (const request of requests) {
    try {
      return await request()
    } catch (error) {
      if (isEndpointFallbackError(error)) {
        fallbackError = error
        continue
      }
      throw error
    }
  }

  throw fallbackError ?? new Error('No supported endpoint available')
}

async function normalizeProfileMutation(payload: Promise<{ data: ProfileMutationResponse }>): Promise<User> {
  const { data } = await payload
  return extractUserFromMutationResponse(data) ?? getProfile()
}

/**
 * Get current user profile
 * @returns User profile data
 */
export async function getProfile(): Promise<User> {
  const { data } = await apiClient.get<User>('/user/profile')
  return data
}

/**
 * Update current user profile
 * @param profile - Profile data to update
 * @returns Updated user profile data
 */
export async function updateProfile(profile: {
  username?: string
  balance_notify_enabled?: boolean
  balance_notify_threshold?: number | null
  balance_notify_extra_emails?: NotifyEmailEntry[]
}): Promise<User> {
  const { data } = await apiClient.put<User>('/user', profile)
  return data
}

/**
 * Change current user password
 * @param passwords - Old and new password
 * @returns Success message
 */
export async function changePassword(
  oldPassword: string,
  newPassword: string
): Promise<{ message: string }> {
  const payload: ChangePasswordRequest = {
    old_password: oldPassword,
    new_password: newPassword
  }

  const { data } = await apiClient.put<{ message: string }>('/user/password', payload)
  return data
}

/**
 * Send verification code for adding a notify email
 * @param email - Email address to verify
 */
export async function sendNotifyEmailCode(email: string): Promise<void> {
  await apiClient.post('/user/notify-email/send-code', { email })
}

/**
 * Verify and add a notify email
 * @param email - Email address to add
 * @param code - Verification code
 */
export async function verifyNotifyEmail(email: string, code: string): Promise<void> {
  await apiClient.post('/user/notify-email/verify', { email, code })
}

/**
 * Remove a notify email
 * @param email - Email address to remove
 */
export async function removeNotifyEmail(email: string): Promise<void> {
  await apiClient.delete('/user/notify-email', { data: { email } })
}

/**
 * Toggle a notify email's disabled state
 * @param email - Email address (empty string for primary email placeholder)
 * @param disabled - Whether to disable the email
 */
export async function toggleNotifyEmail(email: string, disabled: boolean): Promise<User> {
  const { data } = await apiClient.put<User>('/user/notify-email/toggle', { email, disabled })
  return data
}

export async function uploadAvatar(file: File | Blob): Promise<User> {
  const formData = new FormData()
  formData.append('avatar', file, file instanceof File ? file.name : 'avatar.webp')

  return requestFirstSupported([
    () => normalizeProfileMutation(apiClient.post<ProfileMutationResponse>('/user/avatar', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })),
    () => normalizeProfileMutation(apiClient.put<ProfileMutationResponse>('/user/avatar', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })),
    () => normalizeProfileMutation(apiClient.post<ProfileMutationResponse>('/user/profile/avatar', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }))
  ])
}

export async function removeAvatar(): Promise<User> {
  return requestFirstSupported([
    () => normalizeProfileMutation(apiClient.delete<ProfileMutationResponse>('/user/avatar')),
    () => normalizeProfileMutation(apiClient.delete<ProfileMutationResponse>('/user/profile/avatar'))
  ])
}

export async function unbindAccount(provider: UserAccountBindingProvider): Promise<User> {
  return requestFirstSupported([
    () => normalizeProfileMutation(apiClient.delete<ProfileMutationResponse>(`/user/account-bindings/${provider}`)),
    () => normalizeProfileMutation(apiClient.delete<ProfileMutationResponse>(`/user/bindings/${provider}`)),
    () => normalizeProfileMutation(apiClient.delete<ProfileMutationResponse>(`/user/binding/${provider}`)),
    () => normalizeProfileMutation(apiClient.delete<ProfileMutationResponse>('/user/account-bindings', { data: { provider } })),
    () => normalizeProfileMutation(apiClient.delete<ProfileMutationResponse>('/user/bindings', { data: { provider } }))
  ])
}

export function getOAuthBindingStartUrl(
  provider: Extract<UserAccountBindingProvider, 'linuxdo' | 'wechat'>,
  redirectTo: string,
  intent = 'bind'
): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api/v1'
  const normalized = apiBase.replace(/\/$/, '')
  return `${normalized}/auth/oauth/${provider}/start?redirect=${encodeURIComponent(redirectTo)}&intent=${encodeURIComponent(intent)}`
}

export const userAPI = {
  getProfile,
  updateProfile,
  changePassword,
  sendNotifyEmailCode,
  verifyNotifyEmail,
  removeNotifyEmail,
  toggleNotifyEmail,
  uploadAvatar,
  removeAvatar,
  unbindAccount,
  getOAuthBindingStartUrl
}

export default userAPI
