<template>
  <main class="container mx-auto max-w-3xl px-4 py-6">
    <!-- 页面加载状态 -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-20">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
      <span class="mt-4 text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</span>
    </div>

    <!-- 失败页面内容 -->
    <div v-else class="space-y-6">
      <!-- 失败卡片 -->
      <div class="rounded-2xl bg-white p-8 shadow-card dark:bg-dark-800">
        <!-- 失败图标 -->
        <div class="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
          <svg class="h-10 w-10 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </div>

        <!-- 失败标题 -->
        <h1 class="mb-2 text-center text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('recharge.failedTitle') }}
        </h1>

        <!-- 失败原因 -->
        <p class="mb-8 text-center text-gray-500 dark:text-gray-400">
          {{ failureReason }}
        </p>

        <!-- 订单信息 -->
        <div v-if="order" class="mb-8 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('recharge.orderNo') }}</span>
            <span class="font-mono text-gray-900 dark:text-white">{{ orderNo }}</span>
          </div>
          <div class="mt-3 flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('recharge.orderAmount') }}</span>
            <span class="text-gray-900 dark:text-white">¥{{ order.amount?.toFixed(2) || '--' }}</span>
          </div>
          <div class="mt-3 flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('recharge.orderStatus') }}</span>
            <span
              class="rounded-full px-2 py-0.5 text-xs font-medium"
              :class="statusClass"
            >
              {{ statusText }}
            </span>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="flex flex-col gap-3 sm:flex-row">
          <button
            type="button"
            class="btn btn-secondary flex-1 py-3"
            @click="goToDashboard"
          >
            {{ t('recharge.backToDashboard') }}
          </button>
          <button
            type="button"
            class="btn btn-primary flex-1 py-3"
            @click="retryRecharge"
          >
            {{ t('recharge.retryRecharge') }}
          </button>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { rechargeAPI, type RechargeOrder } from '@/api/recharge'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()

// 订单号
const orderNo = computed(() => route.params.orderNo as string)

// 页面加载状态
const loading = ref(true)

// 订单信息
const order = ref<RechargeOrder | null>(null)

// 失败原因
const failureReason = computed(() => {
  if (!order.value) {
    return t('recharge.failedReasonUnknown')
  }
  switch (order.value.status) {
    case 'failed':
      return t('recharge.failedReasonPayment')
    case 'expired':
      return t('recharge.failedReasonExpired')
    case 'cancelled':
      return t('recharge.failedReasonCancelled')
    default:
      return t('recharge.failedReasonUnknown')
  }
})

// 订单状态样式
const statusClass = computed(() => {
  switch (order.value?.status) {
    case 'pending':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
    case 'paid':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    case 'failed':
    case 'expired':
    case 'cancelled':
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
  }
})

// 订单状态文案
const statusText = computed(() => {
  switch (order.value?.status) {
    case 'pending':
      return t('recharge.statusPending')
    case 'paid':
      return t('recharge.statusPaid')
    case 'failed':
      return t('recharge.statusFailed')
    case 'expired':
      return t('recharge.statusExpired')
    case 'cancelled':
      return t('recharge.statusCancelled')
    default:
      return t('recharge.statusUnknown')
  }
})

// 返回仪表盘
const goToDashboard = () => {
  router.push({ name: 'Dashboard' })
}

// 重新充值
const retryRecharge = () => {
  router.push({ name: 'Recharge' })
}

// 加载订单信息
const loadOrder = async () => {
  try {
    const result = await rechargeAPI.getOrder(orderNo.value)
    order.value = result
  } catch (error) {
    console.error('[RechargeFailed] Failed to load order:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadOrder()
})
</script>
