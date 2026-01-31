<template>
  <div class="payment-method-selector">
    <!-- 标签 -->
    <label class="mb-3 block text-sm font-medium text-gray-700 dark:text-gray-300">
      {{ t('recharge.paymentMethod') }}
    </label>

    <!-- 支付方式选项 -->
    <div class="space-y-3">
      <button
        v-for="method in paymentMethods"
        :key="method.id"
        type="button"
        data-testid="payment-method-option"
        :aria-label="t('recharge.selectPaymentMethod', { method: method.name })"
        class="payment-option w-full transition-all duration-200"
        :class="isSelected(method.id) ? 'option-selected' : 'option-default'"
        :disabled="!method.enabled"
        @click="selectMethod(method.id)"
      >
        <div class="flex items-center gap-4">
          <!-- 图标 -->
          <div
            class="flex h-12 w-12 items-center justify-center rounded-xl"
            :class="method.enabled ? method.iconBg : 'bg-gray-100 dark:bg-dark-600'"
          >
            <component
              :is="method.icon"
              class="h-7 w-7"
              :class="method.enabled ? method.iconColor : 'text-gray-400'"
            />
          </div>

          <!-- 名称和描述 -->
          <div class="flex-1 text-left">
            <div
              class="font-medium"
              :class="method.enabled ? 'text-gray-900 dark:text-white' : 'text-gray-400'"
            >
              {{ method.name }}
            </div>
            <div class="text-sm text-gray-500 dark:text-gray-400">
              {{ method.description }}
            </div>
          </div>

          <!-- 选中标记 -->
          <div v-if="isSelected(method.id)" class="flex-shrink-0">
            <svg
              class="h-6 w-6 text-primary-500"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fill-rule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                clip-rule="evenodd"
              />
            </svg>
          </div>

          <!-- 未启用标记 -->
          <div
            v-else-if="!method.enabled"
            class="flex-shrink-0 rounded-full bg-gray-100 px-2 py-1 text-xs text-gray-500 dark:bg-dark-600"
          >
            {{ t('recharge.comingSoonLabel') }}
          </div>
        </div>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useI18n } from 'vue-i18n'

interface Props {
  modelValue: string | null
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: null
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const { t } = useI18n()

// 微信支付图标组件
const WeChatPayIcon = () =>
  h(
    'svg',
    {
      viewBox: '0 0 24 24',
      fill: 'currentColor'
    },
    [
      h('path', {
        d: 'M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 0 1 .213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 0 0 .167-.054l1.903-1.114a.864.864 0 0 1 .717-.098 10.16 10.16 0 0 0 2.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178A1.17 1.17 0 0 1 4.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178 1.17 1.17 0 0 1-1.162-1.178c0-.651.52-1.18 1.162-1.18zm5.34 2.867c-1.797-.052-3.746.512-5.28 1.786-1.72 1.428-2.687 3.72-1.78 6.22.942 2.453 3.666 4.229 6.884 4.229.826 0 1.622-.12 2.361-.336a.722.722 0 0 1 .598.082l1.584.926a.272.272 0 0 0 .14.045c.133 0 .24-.11.24-.245 0-.06-.023-.12-.038-.177l-.327-1.233a.495.495 0 0 1 .179-.555C23.064 18.695 24 17.004 24 15.13c0-3.38-3.099-6.137-7.062-6.272zm-2.813 2.97c.535 0 .969.44.969.982a.976.976 0 0 1-.969.983.976.976 0 0 1-.969-.983c0-.542.434-.982.97-.982zm4.844 0c.536 0 .969.44.969.982a.976.976 0 0 1-.969.983.976.976 0 0 1-.969-.983c0-.542.434-.982.97-.982z'
      })
    ]
  )

// 支付宝图标组件（暂未启用）
const AlipayIcon = () =>
  h(
    'svg',
    {
      viewBox: '0 0 24 24',
      fill: 'currentColor'
    },
    [
      h('path', {
        d: 'M21.422 15.358c-.476-.168-2.139-.746-4.688-1.494a40.24 40.24 0 0 0 1.108-2.49h-4.576v-1.08h5.4v-.72h-5.4V8.268h-1.66c-.132 0-.24.108-.24.24v1.066h-5.4v.72h5.4v1.08H6.39v.72h7.824a25.88 25.88 0 0 1-.744 1.578c-2.37-.554-4.542-.852-6.006-.642-2.94.42-4.944 2.07-4.944 4.206 0 2.346 2.262 3.774 5.712 3.486 2.58-.216 4.692-1.434 6.546-3.348 1.86 1.02 4.908 2.37 6.222 2.988V24H0V0h24v15.816c-.504-.174-1.554-.462-2.578-.458zM6.912 19.266c-2.322.162-3.792-.654-3.792-1.92 0-1.296 1.02-2.466 3.018-2.742 2.34-.324 4.956.438 6.306.78-1.434 2.34-3.264 3.69-5.532 3.882z'
      })
    ]
  )

// 支付方式列表
const paymentMethods = computed(() => [
  {
    id: 'wechat_pay',
    name: t('recharge.wechatPay'),
    description: t('recharge.wechatPayDesc'),
    icon: WeChatPayIcon,
    iconBg: 'bg-green-50 dark:bg-green-900/20',
    iconColor: 'text-green-500',
    enabled: true
  },
  {
    id: 'alipay',
    name: t('recharge.alipay'),
    description: t('recharge.alipayDesc'),
    icon: AlipayIcon,
    iconBg: 'bg-blue-50 dark:bg-blue-900/20',
    iconColor: 'text-blue-500',
    enabled: false
  }
])

// 判断是否选中
const isSelected = (methodId: string): boolean => {
  return props.modelValue === methodId
}

// 选择支付方式
const selectMethod = (methodId: string) => {
  const method = paymentMethods.value.find((m) => m.id === methodId)
  if (method?.enabled) {
    emit('update:modelValue', methodId)
  }
}
</script>

<style scoped>
.payment-option {
  @apply flex items-center rounded-xl border p-4;
}

.option-default {
  @apply border-gray-200 bg-white hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-700 dark:hover:border-dark-500 dark:hover:bg-dark-600;
}

.option-default:disabled {
  @apply cursor-not-allowed opacity-60 hover:border-gray-200 hover:bg-white dark:hover:border-dark-600 dark:hover:bg-dark-700;
}

.option-selected {
  @apply border-primary-500 bg-primary-50 dark:border-primary-400 dark:bg-primary-900/20;
}
</style>
