<template>
  <div
    class="card group relative overflow-hidden transition-all hover:shadow-lg dark:hover:shadow-dark-900/50"
    :class="{
      'ring-2 ring-primary-500': isPopular && !upgradeInfo?.canUpgrade,
      'ring-2 ring-orange-500': upgradeInfo?.canUpgrade
    }"
  >
    <!-- 热门标签 -->
    <div
      v-if="isPopular && !upgradeInfo?.canUpgrade"
      class="absolute right-4 top-4 rounded-full bg-primary-500 px-3 py-1 text-xs font-medium text-white"
    >
      {{ t('subscriptionPlan.popular') }}
    </div>

    <!-- 升级标签 -->
    <div
      v-if="upgradeInfo?.canUpgrade"
      class="absolute right-4 top-4 rounded-full bg-orange-500 px-3 py-1 text-xs font-medium text-white"
    >
      {{ t('subscriptionPlan.upgradeBadge') }}
    </div>

    <div class="p-6">
      <!-- 套餐名称 -->
      <h3 class="mb-2 text-xl font-bold text-gray-900 dark:text-white">
        {{ plan.name }}
      </h3>

      <!-- 套餐描述 -->
      <p
        v-if="plan.purchasable_description || plan.description"
        class="mb-4 text-sm text-gray-500 dark:text-gray-400"
      >
        {{ plan.purchasable_description || plan.description }}
      </p>

      <!-- 价格区域 - 升级模式 -->
      <div v-if="upgradeInfo?.canUpgrade" class="mb-6">
        <div class="flex items-baseline gap-2">
          <span class="text-lg text-gray-400 line-through dark:text-gray-500">¥{{ plan.price_cny }}</span>
          <span class="text-4xl font-bold text-orange-500">¥{{ upgradeInfo.upgradePrice }}</span>
          <span class="text-gray-500 dark:text-gray-400">/月</span>
        </div>
        <p class="mt-1 text-sm text-orange-600 dark:text-orange-400">
          {{ t('subscriptionPlan.upgradeDiscount', { amount: upgradeInfo.remainingValue.toFixed(2) }) }}
        </p>
      </div>

      <!-- 价格区域 - 普通模式 -->
      <div v-else class="mb-6">
        <span class="text-4xl font-bold text-gray-900 dark:text-white">¥{{ plan.price_cny }}</span>
        <span class="text-gray-500 dark:text-gray-400">/月</span>
      </div>

      <!-- 额度信息 -->
      <div class="mb-6 space-y-2">
        <div v-if="plan.daily_limit_usd" class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
          <svg class="h-4 w-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
          <span>{{ t('subscriptionPlan.dailyQuota', { amount: plan.daily_limit_usd }) }}</span>
        </div>
        <div v-if="plan.weekly_limit_usd" class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
          <svg class="h-4 w-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
          <span>{{ t('subscriptionPlan.weeklyQuota', { amount: plan.weekly_limit_usd }) }}</span>
        </div>
        <div v-if="plan.monthly_limit_usd" class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
          <svg class="h-4 w-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
          <span>{{ t('subscriptionPlan.monthlyQuota', { amount: plan.monthly_limit_usd }) }}</span>
        </div>
        <div v-if="!plan.daily_limit_usd && !plan.weekly_limit_usd && !plan.monthly_limit_usd" class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
          <svg class="h-4 w-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
          <span>{{ t('subscriptionPlan.unlimitedQuota') }}</span>
        </div>
      </div>

      <!-- 已订阅按钮（无升级选项时） -->
      <button
        v-if="subscribed && !upgradeInfo?.canUpgrade"
        type="button"
        class="btn w-full cursor-not-allowed border-green-200 bg-green-50 text-green-600 dark:border-green-800 dark:bg-green-900/20 dark:text-green-400"
        disabled
      >
        <svg class="mr-1.5 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
        </svg>
        {{ t('subscriptionPlan.subscribed') }}
      </button>

      <!-- 升级按钮 -->
      <button
        v-else-if="upgradeInfo?.canUpgrade"
        type="button"
        class="btn w-full border-orange-300 bg-orange-500 text-white hover:bg-orange-600 dark:border-orange-600 dark:bg-orange-600 dark:hover:bg-orange-700"
        :disabled="loading"
        @click="handlePurchase"
      >
        <span v-if="loading" class="flex items-center justify-center gap-2">
          <div class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"></div>
          {{ t('common.loading') }}
        </span>
        <span v-else>{{ t('subscriptionPlan.upgradeNow') }}</span>
      </button>

      <!-- 普通购买按钮 -->
      <button
        v-else
        type="button"
        class="btn w-full"
        :class="isPopular ? 'btn-primary' : 'btn-outline'"
        :disabled="loading"
        @click="handlePurchase"
      >
        <span v-if="loading" class="flex items-center justify-center gap-2">
          <div class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"></div>
          {{ t('common.loading') }}
        </span>
        <span v-else>{{ t('subscriptionPlan.purchase') }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/api/subscriptionPlan'

export interface UpgradeInfo {
  canUpgrade: boolean
  sourceSubscriptionId: number
  upgradePrice: number
  originalPrice: number
  remainingValue: number
}

const props = defineProps<{
  plan: SubscriptionPlan
  isPopular?: boolean
  subscribed?: boolean
  upgradeInfo?: UpgradeInfo
}>()

const emit = defineEmits<{
  purchase: [planId: number]
}>()

const { t } = useI18n()
const loading = ref(false)

const handlePurchase = () => {
  emit('purchase', props.plan.id)
}

defineExpose({
  setLoading: (value: boolean) => {
    loading.value = value
  }
})
</script>
