<template>
  <AppLayout>
    <div class="mx-auto max-w-2xl space-y-6">
      <!-- Page Title -->
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
        {{ t('lottery.coupons.title') }}
      </h1>

      <!-- Status Filter Tabs -->
      <div class="flex gap-2">
        <button
          v-for="tab in statusTabs"
          :key="tab.value"
          :class="[
            'rounded-full px-4 py-1.5 text-sm font-medium transition-colors',
            currentStatus === tab.value
              ? 'bg-primary-500 text-white'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-dark-400 dark:hover:bg-dark-600'
          ]"
          @click="changeStatus(tab.value)"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else>
        <!-- Coupon List -->
        <div v-if="coupons.length > 0" class="space-y-4">
          <div
            v-for="coupon in coupons"
            :key="coupon.id"
            class="card overflow-hidden"
          >
            <div class="flex">
              <!-- Left Color Stripe -->
              <div
                :class="[
                  'w-2 flex-shrink-0',
                  coupon.coupon_type === 'winner_discount' ? 'bg-green-500' : 'bg-blue-500'
                ]"
              ></div>

              <!-- Coupon Content -->
              <div class="flex flex-1 items-center justify-between p-4">
                <div class="min-w-0 flex-1 space-y-2">
                  <!-- Type & Value -->
                  <div class="flex items-center gap-2">
                    <span
                      :class="[
                        'badge',
                        coupon.coupon_type === 'winner_discount' ? 'badge-success' : 'badge-info'
                      ]"
                    >
                      {{ t(`lottery.coupons.type.${coupon.coupon_type}`) }}
                    </span>
                    <span class="text-lg font-bold text-gray-900 dark:text-white">
                      <template v-if="coupon.coupon_type === 'winner_discount' && coupon.discount_percent">
                        {{ coupon.discount_percent }}% {{ t('lottery.coupons.discount', { percent: coupon.discount_percent }) }}
                      </template>
                      <template v-else-if="coupon.reduction_amount">
                        {{ t('lottery.coupons.reduction', { amount: coupon.reduction_amount }) }}
                      </template>
                    </span>
                  </div>

                  <!-- Scope -->
                  <p class="text-xs text-gray-500 dark:text-dark-400">
                    {{ t(`lottery.coupons.scope.${coupon.applicable_scope}`) }}
                  </p>

                  <!-- Dates -->
                  <div class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-400 dark:text-dark-500">
                    <span>{{ t('lottery.coupons.validUntil') }}: {{ formatDateTime(coupon.expires_at) }}</span>
                    <span v-if="coupon.used_at">{{ t('lottery.coupons.usedAt') }}: {{ formatDateTime(coupon.used_at) }}</span>
                  </div>
                </div>

                <!-- Status Badge -->
                <div class="ml-4 flex-shrink-0">
                  <span
                    :class="[
                      'badge',
                      coupon.status === 'active'
                        ? 'badge-success'
                        : coupon.status === 'used'
                          ? 'badge-secondary'
                          : 'badge-danger'
                    ]"
                  >
                    {{ t(`lottery.coupons.status.${coupon.status}`) }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- Pagination -->
          <div
            v-if="totalPages > 1"
            class="flex items-center justify-center gap-2 pt-4"
          >
            <button
              class="btn btn-secondary text-sm"
              :disabled="currentPage <= 1"
              @click="loadCoupons(currentPage - 1)"
            >
              &laquo;
            </button>
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ currentPage }} / {{ totalPages }}
            </span>
            <button
              class="btn btn-secondary text-sm"
              :disabled="currentPage >= totalPages"
              @click="loadCoupons(currentPage + 1)"
            >
              &raquo;
            </button>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="card p-12 text-center">
          <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
            <svg class="h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" />
            </svg>
          </div>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            {{ t('lottery.coupons.noCoupons') }}
          </p>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getMyCoupons } from '@/api/lottery'
import type { LotteryCoupon } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()

const coupons = ref<LotteryCoupon[]>([])
const loading = ref(true)
const currentPage = ref(1)
const totalPages = ref(1)
const currentStatus = ref('')

const statusTabs = computed(() => [
  { value: '', label: t('lottery.coupons.status.all') },
  { value: 'active', label: t('lottery.coupons.status.active') },
  { value: 'used', label: t('lottery.coupons.status.used') },
  { value: 'expired', label: t('lottery.coupons.status.expired') }
])

function changeStatus(status: string) {
  currentStatus.value = status
  loadCoupons(1)
}

async function loadCoupons(page: number = 1) {
  loading.value = true
  try {
    const res = await getMyCoupons(page, 20, currentStatus.value || undefined)
    coupons.value = res.items
    currentPage.value = res.page
    totalPages.value = res.pages
  } catch (error) {
    console.error('Failed to load coupons:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => loadCoupons())
</script>
