/**
 * Admin Model Pricing API endpoints
 * Manages fallback billing prices for models
 */

import { apiClient } from '../client'

export interface ModelPricingEntry {
  id: number
  model_key: string
  display_name: string
  input_price_per_million: number
  output_price_per_million: number
  input_price_per_million_priority: number
  output_price_per_million_priority: number
  cache_read_price_per_million: number
  cache_read_price_per_million_priority: number
  cache_creation_price_per_million: number
  enabled: boolean
  note: string
  created_at: string
  updated_at: string
}

export interface UpsertModelPricingRequest {
  model_key: string
  display_name?: string
  input_price_per_million: number
  output_price_per_million: number
  input_price_per_million_priority: number
  output_price_per_million_priority: number
  cache_read_price_per_million: number
  cache_read_price_per_million_priority: number
  cache_creation_price_per_million: number
  enabled: boolean
  note?: string
}

export async function list(): Promise<ModelPricingEntry[]> {
  const { data } = await apiClient.get<ModelPricingEntry[]>('/admin/model-pricings')
  return data
}

export async function create(req: UpsertModelPricingRequest): Promise<ModelPricingEntry> {
  const { data } = await apiClient.post<ModelPricingEntry>('/admin/model-pricings', req)
  return data
}

export async function update(id: number, req: UpsertModelPricingRequest): Promise<ModelPricingEntry> {
  const { data } = await apiClient.put<ModelPricingEntry>(`/admin/model-pricings/${id}`, req)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/model-pricings/${id}`)
}

export const modelPricingsAPI = { list, create, update, remove }
export default modelPricingsAPI
