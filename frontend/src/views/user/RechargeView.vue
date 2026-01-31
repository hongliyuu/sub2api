<template>
  <main class="container mx-auto max-w-3xl px-4 py-6">
    <!-- 页面加载状态 -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-20">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent"></div>
      <span class="mt-4 text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</span>
    </div>

    <!-- 主体内容 -->
    <div v-else class="space-y-6">
      <!-- 余额展示卡片 -->
      <div
        class="balance-card bg-gradient-to-r from-primary-500 to-primary-600 rounded-2xl p-10 text-center text-white shadow-lg"
      >
        <!-- 钱包图标 -->
        <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-white/20">
          <svg class="h-8 w-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M21 12a2.25 2.25 0 00-2.25-2.25H15a3 3 0 11-6 0H5.25A2.25 2.25 0 003 12m18 0v6a2.25 2.25 0 01-2.25 2.25H5.25A2.25 2.25 0 013 18v-6m18 0V9M3 12V9m18 0a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 9m18 0V6a2.25 2.25 0 00-2.25-2.25H5.25A2.25 2.25 0 003 6v3"
            />
          </svg>
        </div>
        <!-- 余额标签 -->
        <span class="block text-sm opacity-80">{{ t('recharge.currentBalance') }}</span>
        <!-- 余额数值 -->
        <span class="mt-2 block text-5xl font-bold">¥{{ formattedBalance }}</span>
      </div>

      <!-- 充值表单区域 -->
      <div class="rounded-2xl bg-white p-6 shadow-card dark:bg-dark-800">
        <h2 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('recharge.title') }}
        </h2>
        <p class="mb-6 text-sm text-gray-500 dark:text-gray-400">
          {{ t('recharge.subtitle') }}
        </p>

        <!-- 金额选择器 -->
        <AmountSelector
          v-model="selectedAmount"
          :default-amounts="rechargeStore.defaultAmounts"
          :min-amount="rechargeStore.minAmount"
          :max-amount="rechargeStore.maxAmount"
        />

        <!-- 支付方式选择器 -->
        <div class="mt-6">
          <PaymentMethodSelector v-model="selectedPaymentMethod" />
        </div>

        <!-- 错误提示 -->
        <div
          v-if="errorMessage"
          class="mt-4 rounded-lg bg-red-50 p-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-400"
        >
          {{ errorMessage }}
        </div>

        <!-- 提交按钮 -->
        <div class="mt-6">
          <button
            type="button"
            :disabled="!canSubmit || submitting"
            :aria-label="submitButtonText"
            class="btn btn-primary w-full py-3 text-base font-medium transition-all duration-200"
            :class="{
              'opacity-50 cursor-not-allowed': !canSubmit || submitting,
              'hover:shadow-lg': canSubmit && !submitting
            }"
            @click="handleSubmit"
          >
            <span v-if="submitting" class="flex items-center justify-center gap-2">
              <svg class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                />
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
              {{ t('recharge.orderCreating') }}
            </span>
            <span v-else>{{ submitButtonText }}</span>
          </button>
        </div>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore, useRechargeStore } from '@/stores'
import { rechargeAPI } from '@/api'
import AmountSelector from '@/components/user/recharge/AmountSelector.vue'
import PaymentMethodSelector from '@/components/user/recharge/PaymentMethodSelector.vue'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const rechargeStore = useRechargeStore()

// 页面加载状态
const loading = ref(true)

// 选中的充值金额
const selectedAmount = ref<number | null>(null)

// 选中的支付方式
const selectedPaymentMethod = ref<string | null>(null)

// 提交中状态
const submitting = ref(false)

// 错误信息
const errorMessage = ref('')

// 用户余额
const balance = computed(() => authStore.user?.balance ?? 0)

// 格式化余额显示（保留两位小数）
const formattedBalance = computed(() => balance.value.toFixed(2))

// 金额有效性验证
const isAmountValid = computed(() => {
  if (selectedAmount.value === null) {
    return false
  }
  const amount = selectedAmount.value
  return amount >= rechargeStore.minAmount && amount <= rechargeStore.maxAmount
})

// 支付方式有效性验证
const isPaymentMethodValid = computed(() => {
  return selectedPaymentMethod.value !== null
})

// 是否可以提交
const canSubmit = computed(() => {
  return isAmountValid.value && isPaymentMethodValid.value
})

// 提交按钮文案
const submitButtonText = computed(() => {
  if (!isPaymentMethodValid.value) {
    return t('recharge.selectPaymentMethodFirst')
  }
  if (selectedAmount.value !== null && isAmountValid.value) {
    return t('recharge.submitButton', { amount: selectedAmount.value })
  }
  return t('recharge.submitButtonDefault')
})

// 提交处理
const handleSubmit = async () => {
  if (!canSubmit.value || submitting.value) {
    return
  }

  // 清除之前的错误
  errorMessage.value = ''
  submitting.value = true

  try {
    // 调用 API 创建订单
    const order = await rechargeAPI.createOrder({
      amount: selectedAmount.value!,
      payment_method: selectedPaymentMethod.value!
    })

    // 订单创建成功，跳转到支付页面
    router.push({
      name: 'RechargePayment',
      params: { orderNo: order.order_no }
    })
  } catch (error) {
    console.error('Failed to create order:', error)
    errorMessage.value = t('recharge.orderCreateFailed')
  } finally {
    submitting.value = false
  }
}

// 页面加载时刷新用户数据以获取最新余额
onMounted(async () => {
  try {
    // 并行加载用户数据和充值配置
    await Promise.all([authStore.refreshUser(), rechargeStore.fetchConfig()])
  } catch (error) {
    console.error('Failed to refresh user data:', error)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
/* 余额卡片渐变背景 */
.balance-card {
  background: linear-gradient(135deg, #d97757 0%, #c45a3a 100%);
}
</style>
