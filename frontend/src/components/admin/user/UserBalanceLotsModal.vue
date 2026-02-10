<template>
  <BaseDialog :show="show" :title="t('admin.users.balanceLotsTitle')" width="wide" :close-on-click-outside="true" :z-index="40" @close="$emit('close')">
    <div v-if="user" class="space-y-4">
      <!-- User header -->
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
            <span class="text-lg font-medium text-primary-700 dark:text-primary-300">
              {{ user.email.charAt(0).toUpperCase() }}
            </span>
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
            <p class="text-xs text-gray-400 dark:text-dark-500">
              {{ t('admin.users.currentBalance') }}: ${{ user.balance?.toFixed(2) || '0.00' }}
            </p>
          </div>
        </div>
      </div>

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
        <p class="text-sm text-gray-500">{{ t('admin.users.noBalanceLots') }}</p>
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
                  <span v-if="lot.source_ref" class="ml-1 font-mono text-xs text-gray-400">{{ lot.source_ref.slice(0, 12) }}</span>
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
                  {{ t('admin.users.balanceLotExpiredAt') }}: {{ formatDateTime(lot.expired_at || lot.expires_at) }}
                </template>
                <template v-else>
                  {{ t('admin.users.balanceLotExpiresAt') }}: {{ formatDateTime(lot.expires_at) }}
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
import { adminAPI } from '@/api/admin'
import { formatDateTime } from '@/utils/format'
import type { AdminUser, BalanceLot, BalanceLotStatus, BalanceLotSourceType } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
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
  { value: '', label: t('admin.users.allStatuses') },
  { value: 'active', label: t('admin.users.balanceLotActive') },
  { value: 'depleted', label: t('admin.users.balanceLotDepleted') },
  { value: 'expired', label: t('admin.users.balanceLotExpired') }
])

watch(() => props.show, (v) => {
  if (v && props.user) {
    statusFilter.value = ''
    loadLots(1)
  }
})

const loadLots = async (page: number) => {
  if (!props.user) return
  loading.value = true
  currentPage.value = page
  try {
    const res = await adminAPI.users.getUserBalanceLots(props.user.id, page, pageSize, statusFilter.value || undefined)
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
    case 'recharge': return t('admin.users.balanceLotSourceRecharge')
    case 'redeem': return t('admin.users.balanceLotSourceRedeem')
    case 'promo': return t('admin.users.balanceLotSourcePromo')
    case 'adjust': return t('admin.users.balanceLotSourceAdjust')
    case 'migration': return t('admin.users.balanceLotSourceMigration')
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
    case 'active': return t('admin.users.balanceLotActive')
    case 'depleted': return t('admin.users.balanceLotDepleted')
    case 'expired': return t('admin.users.balanceLotExpired')
    default: return status
  }
}
</script>
