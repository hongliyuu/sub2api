/**
 * Lottery API endpoints
 * User-facing lottery participation and coupon management
 */

import { apiClient } from './client'
import type {
  LotteryActivity,
  LotteryParticipant,
  LotteryCoupon,
  BasePaginationResponse
} from '@/types'

/**
 * Get currently active lottery activities
 */
export async function getActive(): Promise<LotteryActivity[]> {
  const { data } = await apiClient.get<LotteryActivity[]>('/lottery/active')
  return data
}

/**
 * Get user's participation history
 */
export async function getMyParticipations(
  page: number = 1,
  pageSize: number = 20
): Promise<BasePaginationResponse<LotteryParticipant>> {
  const { data } = await apiClient.get<BasePaginationResponse<LotteryParticipant>>('/lottery/my', {
    params: { page, page_size: pageSize }
  })
  return data
}

/**
 * Get user's lottery coupons
 */
export async function getMyCoupons(
  page: number = 1,
  pageSize: number = 20,
  status?: string
): Promise<BasePaginationResponse<LotteryCoupon>> {
  const { data } = await apiClient.get<BasePaginationResponse<LotteryCoupon>>('/lottery/coupons', {
    params: { page, page_size: pageSize, status }
  })
  return data
}

/**
 * Get lottery activity details by ID
 */
export async function getById(id: number): Promise<LotteryActivity> {
  const { data } = await apiClient.get<LotteryActivity>(`/lottery/${id}`)
  return data
}

/**
 * Participate in a lottery activity
 */
export async function participate(id: number): Promise<LotteryParticipant> {
  const { data } = await apiClient.post<LotteryParticipant>(`/lottery/${id}/participate`)
  return data
}

/**
 * Get lottery activity by share code
 */
export async function getByShareCode(code: string): Promise<LotteryActivity> {
  const { data } = await apiClient.get<LotteryActivity>(`/lottery/share/${code}`)
  return data
}

export const lotteryAPI = {
  getActive,
  getMyParticipations,
  getMyCoupons,
  getById,
  participate,
  getByShareCode
}

export default lotteryAPI
