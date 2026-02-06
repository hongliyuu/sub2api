/**
 * User API endpoints
 * Handles user profile management and password changes
 */

import { apiClient } from './client'
import type { User, ChangePasswordRequest } from '@/types'

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
 * Send verification code for setting password (OAuth users only)
 * @returns Success message with countdown
 */
export async function sendSetPasswordCode(): Promise<{ message: string; countdown: number }> {
  const { data } = await apiClient.post<{ message: string; countdown: number }>('/user/send-set-password-code')
  return data
}

/**
 * Set password for OAuth users who don't have one
 * Requires email verification code for identity verification
 * @param email - User's email address
 * @param verifyCode - Email verification code
 * @param newPassword - New password to set
 * @returns Success message
 */
export async function setPassword(
  email: string,
  verifyCode: string,
  newPassword: string
): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>('/user/set-password', {
    email,
    verify_code: verifyCode,
    new_password: newPassword
  })
  return data
}

export const userAPI = {
  getProfile,
  updateProfile,
  changePassword,
  sendSetPasswordCode,
  setPassword
}

export default userAPI
