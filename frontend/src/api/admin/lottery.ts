/**
 * Admin Lottery API endpoints
 */

import { apiClient } from '../client'
import type {
  LotteryActivity,
  LotteryParticipant,
  LotteryDrawResult,
  CreateLotteryActivityRequest,
  BasePaginationResponse
} from '@/types'

export async function create(request: CreateLotteryActivityRequest): Promise<LotteryActivity> {
  const { data } = await apiClient.post<LotteryActivity>('/admin/lottery', request)
  return data
}

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: string
  }
): Promise<BasePaginationResponse<LotteryActivity>> {
  const { data } = await apiClient.get<BasePaginationResponse<LotteryActivity>>('/admin/lottery', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function getById(id: number): Promise<LotteryActivity> {
  const { data } = await apiClient.get<LotteryActivity>(`/admin/lottery/${id}`)
  return data
}

export async function draw(id: number): Promise<LotteryDrawResult> {
  const { data } = await apiClient.post<LotteryDrawResult>(`/admin/lottery/${id}/draw`)
  return data
}

export async function cancel(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/lottery/${id}`)
  return data
}

export async function listParticipants(
  id: number,
  page: number = 1,
  pageSize: number = 20
): Promise<BasePaginationResponse<LotteryParticipant>> {
  const { data } = await apiClient.get<BasePaginationResponse<LotteryParticipant>>(
    `/admin/lottery/${id}/participants`,
    { params: { page, page_size: pageSize } }
  )
  return data
}

const lotteryAPI = {
  create,
  list,
  getById,
  draw,
  cancel,
  listParticipants
}

export default lotteryAPI
