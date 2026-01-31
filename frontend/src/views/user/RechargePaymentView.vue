<template>
  <main class="container mx-auto max-w-3xl px-4 py-6">
    <!-- 页面加载状态 -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-20">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
      <span class="mt-4 text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</span>
    </div>

    <!-- 支付页面内容 -->
    <div v-else class="space-y-6">
      <!-- 返回按钮 -->
      <button
        type="button"
        class="flex items-center gap-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
        @click="goBack"
      >
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
        </svg>
        {{ t('common.back') }}
      </button>

      <!-- 订单信息卡片 -->
      <div class="rounded-2xl bg-white p-6 shadow-card dark:bg-dark-800">
        <h2 class="mb-4 text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('recharge.orderInfo') }}
        </h2>

        <!-- 订单详情 -->
        <div class="space-y-3">
          <div class="flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('recharge.orderNo') }}</span>
            <span class="font-mono text-gray-900 dark:text-white">{{ orderNo }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('recharge.orderAmount') }}</span>
            <span class="text-lg font-semibold text-primary-500">¥{{ order?.amount?.toFixed(2) || '--' }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t('recharge.orderStatus') }}</span>
            <span
              class="rounded-full px-2 py-0.5 text-xs font-medium"
              :class="statusClass"
            >
              {{ statusText }}
            </span>
          </div>
        </div>

        <!-- 支付二维码区域（后续 Story 实现） -->
        <div class="mt-6 flex flex-col items-center justify-center rounded-xl bg-gray-50 p-8 dark:bg-dark-700">
          <div class="mb-4 text-center text-gray-500 dark:text-gray-400">
            {{ t('recharge.qrcodeComingSoon') }}
          </div>
          <div class="h-48 w-48 rounded-lg bg-gray-200 dark:bg-dark-600"></div>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import type { RechargeOrder } from '@/api/recharge'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()

// 订单号
const orderNo = computed(() => route.params.orderNo as string)

// 页面加载状态
const loading = ref(true)

// 订单信息
const order = ref<RechargeOrder | null>(null)

// 订单状态样式
const statusClass = computed(() => {
  switch (order.value?.status) {
    case 'pending':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
    case 'paid':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    case 'failed':
    case 'expired':
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
    default:
      return t('recharge.statusUnknown')
  }
})

// 返回上一页
const goBack = () => {
  router.push({ name: 'Recharge' })
}

// 加载订单信息
onMounted(async () => {
  try {
    // TODO: Story 2-6/2-7 实现订单查询和二维码展示
    // 目前使用模拟数据
    order.value = {
      id: 1,
      order_no: orderNo.value,
      amount: 100,
      status: 'pending',
      payment_method: 'wechat_pay',
      payment_channel: 'native',
      created_at: new Date().toISOString(),
      expire_at: new Date(Date.now() + 30 * 60 * 1000).toISOString()
    }
  } catch (error) {
    console.error('Failed to load order:', error)
  } finally {
    loading.value = false
  }
})
</script>
