/**
 * Balance Lots API endpoints
 * User-facing balance lot (余额批次) operations
 */

import { apiClient } from './client'
import type { BalanceLot, BalanceLotSummary, PaginatedResponse } from '@/types'

export async function getUserBalanceLots(
  page: number = 1,
  pageSize: number = 20,
  status?: string,
): Promise<PaginatedResponse<BalanceLot>> {
  const params: Record<string, unknown> = { page, page_size: pageSize }
  if (status) {
    params.status = status
  }
  const { data } = await apiClient.get<PaginatedResponse<BalanceLot>>('/balance-lots', {
    params,
  })
  return data
}

export async function getUserBalanceLotsSummary(): Promise<BalanceLotSummary> {
  const { data } = await apiClient.get<BalanceLotSummary>('/balance-lots/summary')
  return data
}

export const balanceLotsAPI = {
  list: getUserBalanceLots,
  summary: getUserBalanceLotsSummary,
}

export default balanceLotsAPI
