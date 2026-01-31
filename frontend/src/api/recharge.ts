/**
 * Recharge API Client
 * Handles balance recharge configuration and operations
 */

import { apiClient } from './client'

// ==================== Types ====================

export interface RechargeConfig {
  enabled: boolean
  min_amount: number
  max_amount: number
  default_amounts: number[]
}

export interface CreateOrderRequest {
  amount: number
  payment_method: string
}

export interface RechargeOrder {
  id: number
  order_no: string
  amount: number
  status: string
  payment_method: string
  payment_channel: string
  created_at: string
  expire_at: string
}

// ==================== API Functions ====================

export const rechargeAPI = {
  /**
   * 获取充值配置（公开接口，无需认证）
   */
  async getConfig(): Promise<RechargeConfig> {
    const response = await apiClient.get<RechargeConfig>('/recharge/config')
    return response.data
  },

  /**
   * 创建充值订单
   */
  async createOrder(data: CreateOrderRequest): Promise<RechargeOrder> {
    const response = await apiClient.post<RechargeOrder>('/recharge/orders', data)
    return response.data
  }
}

export default rechargeAPI
