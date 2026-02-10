<template>
  <BaseDialog :show="show" :title="t('balanceLots.title')" width="wide" :close-on-click-outside="true" :z-index="40" @close="$emit('close')">
    <div class="space-y-4">
      <!-- Status filter -->
      <div class="flex items-center gap-3">
        <Select
          v-model="statusFilter"
          :options="statusOptions"
          class="w-48"
          @change="loadLots(1)"
        />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-8">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>

      <!-- Empty state -->
      <div v-else-if="lots.length === 0" class="py-8 text-center">
        <p class="text-sm text-gray-500">{{ t('balanceLots.noLots') }}</p>
      </div>

      <!-- Lots list -->
      <div v-else class="max-h-[28rem] space-y-3 overflow-y-auto">
        <div
          v-for="lot in lots"
          :key="lot.id"
          class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="flex items-start justify-between">
            <!-- Left: source info -->
            <div class="flex items-start gap-3">
              <div :class="['flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg', getStatusBg(lot.status)]">
                <Icon :name="getSourceIcon(lot.source_type) as any" size="sm" :class="getStatusColor(lot.status)" />
              </div>
              <div>
                <p class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ getSourceLabel(lot.source_type) }}
                </p>
                <p v-if="lot.description" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400" :title="lot.description">
                  {{ lot.description.length > 50 ? lot.description.substring(0, 45) + '...' : lot.description }}
                </p>
                <p class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">
                  {{ formatDateTime(lot.created_at) }}
                </p>
              </div>
            </div>
            <!-- Right: amounts and expiry -->
            <div class="text-right">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">
                ${{ lot.remaining_amount.toFixed(2) }}
                <span v-if="lot.remaining_amount !== lot.original_amount" class="text-xs font-normal text-gray-400">
                  / ${{ lot.original_amount.toFixed(2) }}
                </span>
              </p>
              <span :class="['inline-block rounded-full px-2 py-0.5 text-xs font-medium', getStatusBadge(lot.status)]">
                {{ getStatusLabel(lot.status) }}
              </span>
              <p class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">
                <template v-if="lot.status === 'expired'">
                  {{ t('balanceLots.expiredAt') }}: {{ formatDateTime(lot.expired_at || lot.expires_at) }}
                </template>
                <template v-else>
                  {{ t('balanceLots.expiresAt') }}: {{ formatDateTime(lot.expires_at) }}
                </template>
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-center gap-2 pt-2">
        <button
          :disabled="currentPage <= 1"
          class="btn btn-secondary px-3 py-1 text-sm"
          @click="loadLots(currentPage - 1)"
        >
          {{ t('pagination.previous') }}
        </button>
        <span class="text-sm text-gray-500 dark:text-dark-400">
          {{ currentPage }} / {{ totalPages }}
        </span>
        <button
          :disabled="currentPage >= totalPages"
          class="btn btn-secondary px-3 py-1 text-sm"
          @click="loadLots(currentPage + 1)"
        >
          {{ t('pagination.next') }}
        </button>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { balanceLotsAPI } from '@/api/balanceLots'
import { formatDateTime } from '@/utils/format'
import type { BalanceLot, BalanceLotStatus, BalanceLotSourceType } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ show: boolean }>()
defineEmits(['close'])
const { t } = useI18n()

const lots = ref<BalanceLot[]>([])
const loading = ref(false)
const currentPage = ref(1)
const total = ref(0)
const pageSize = 15
const statusFilter = ref('')

const totalPages = computed(() => Math.ceil(total.value / pageSize) || 1)

const statusOptions = computed(() => [
  { value: '', label: t('balanceLots.allStatuses') },
  { value: 'active', label: t('balanceLots.active') },
  { value: 'depleted', label: t('balanceLots.depleted') },
  { value: 'expired', label: t('balanceLots.expired') }
])

watch(() => props.show, (v) => {
  if (v) {
    statusFilter.value = ''
    loadLots(1)
  }
})

const loadLots = async (page: number) => {
  loading.value = true
  currentPage.value = page
  try {
    const res = await balanceLotsAPI.list(page, pageSize, statusFilter.value || undefined)
    lots.value = res.items || []
    total.value = res.total || 0
  } catch (error) {
    console.error('Failed to load balance lots:', error)
  } finally {
    loading.value = false
  }
}

const getSourceIcon = (source: BalanceLotSourceType): string => {
  switch (source) {
    case 'recharge': return 'dollar'
    case 'redeem': return 'badge'
    case 'promo': return 'badge'
    case 'adjust': return 'edit'
    case 'migration': return 'database'
    default: return 'dollar'
  }
}

const getSourceLabel = (source: BalanceLotSourceType): string => {
  switch (source) {
    case 'recharge': return t('balanceLots.sourceRecharge')
    case 'redeem': return t('balanceLots.sourceRedeem')
    case 'promo': return t('balanceLots.sourcePromo')
    case 'adjust': return t('balanceLots.sourceAdjust')
    case 'migration': return t('balanceLots.sourceMigration')
    default: return source
  }
}

const getStatusBg = (status: BalanceLotStatus): string => {
  switch (status) {
    case 'active': return 'bg-emerald-100 dark:bg-emerald-900/30'
    case 'depleted': return 'bg-gray-100 dark:bg-gray-800'
    case 'expired': return 'bg-red-100 dark:bg-red-900/30'
    default: return 'bg-gray-100 dark:bg-gray-800'
  }
}

const getStatusColor = (status: BalanceLotStatus): string => {
  switch (status) {
    case 'active': return 'text-emerald-600 dark:text-emerald-400'
    case 'depleted': return 'text-gray-400 dark:text-gray-500'
    case 'expired': return 'text-red-600 dark:text-red-400'
    default: return 'text-gray-400'
  }
}

const getStatusBadge = (status: BalanceLotStatus): string => {
  switch (status) {
    case 'active': return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
    case 'depleted': return 'bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400'
    case 'expired': return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    default: return 'bg-gray-100 text-gray-500'
  }
}

const getStatusLabel = (status: BalanceLotStatus): string => {
  switch (status) {
    case 'active': return t('balanceLots.active')
    case 'depleted': return t('balanceLots.depleted')
    case 'expired': return t('balanceLots.expired')
    default: return status
  }
}
</script>
