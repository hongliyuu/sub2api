/**
 * Admin Balance API endpoints
 * Handles balance group user statistics for administrators
 */

import { apiClient } from '../client'

/** Aggregated usage statistics for a single user in a balance group */
export interface BalanceGroupUserStats {
  user_id: number
  email: string
  username: string
  balance: number
  total_cost: number
  actual_cost: number
  total_requests: number
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
}

/** Paginated response for balance group user stats */
export interface BalanceGroupUserStatsResponse {
  users: BalanceGroupUserStats[]
  total: number
}

/** Query parameters for balance group user stats */
export interface BalanceGroupUserStatsParams {
  group_id: number
  start_date: string
  end_date: string
  page?: number
  page_size?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  search?: string
}

/**
 * Get balance group user statistics
 * @param params - Query parameters
 * @returns Paginated user stats response
 */
export async function getBalanceGroupUserStats(
  params: BalanceGroupUserStatsParams,
  options?: { signal?: AbortSignal }
): Promise<BalanceGroupUserStatsResponse> {
  const { data } = await apiClient.get<BalanceGroupUserStatsResponse>(
    '/admin/balance/stats',
    {
      params,
      signal: options?.signal
    }
  )
  return data
}

export default {
  getBalanceGroupUserStats
}
