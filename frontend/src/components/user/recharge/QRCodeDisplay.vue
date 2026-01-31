<template>
  <div class="qrcode-display flex flex-col items-center justify-center">
    <!-- 错误状态 -->
    <div v-if="error" class="flex flex-col items-center text-center" data-testid="error-state">
      <svg class="h-12 w-12 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <span class="mt-3 text-sm text-red-500">{{ t('recharge.qrcodeError') }}</span>
      <button
        type="button"
        class="mt-3 text-sm text-primary-500 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-50"
        :disabled="retryCount >= MAX_RETRIES"
        :aria-label="t('common.retry')"
        @click="retryGenerate"
      >
        {{ retryCount >= MAX_RETRIES ? t('recharge.maxRetriesReached') : t('common.retry') }}
      </button>
    </div>

    <!-- 二维码展示 -->
    <div v-else class="flex flex-col items-center" data-testid="qrcode-state">
      <!-- 二维码画布 -->
      <div
        class="rounded-xl bg-white p-4 shadow-md dark:bg-gray-100"
        :style="{ width: `${size + 32}px`, height: `${size + 32}px` }"
      >
        <canvas
          ref="canvasRef"
          :width="size"
          :height="size"
          data-testid="qrcode-canvas"
        ></canvas>
      </div>

      <!-- 扫码提示 -->
      <div class="mt-4 flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
        <!-- 微信图标 -->
        <svg class="h-5 w-5 text-green-500" viewBox="0 0 24 24" fill="currentColor">
          <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 01.213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 00.167-.054l1.903-1.114a.864.864 0 01.717-.098 10.16 10.16 0 002.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178A1.17 1.17 0 014.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 01-1.162 1.178 1.17 1.17 0 01-1.162-1.178c0-.651.52-1.18 1.162-1.18zm5.34 2.867c-1.797-.052-3.746.512-5.28 1.786-1.72 1.428-2.687 3.72-1.78 6.22.942 2.453 3.666 4.229 6.884 4.229.826 0 1.622-.12 2.361-.336a.722.722 0 01.598.082l1.584.926a.272.272 0 00.14.047c.134 0 .24-.111.24-.247 0-.06-.023-.12-.038-.177l-.327-1.233a.582.582 0 01-.023-.156.49.49 0 01.201-.398C23.024 18.48 24 16.82 24 14.98c0-3.21-2.931-5.837-7.062-6.122zm-2.036 2.533c.535 0 .969.44.969.982a.976.976 0 01-.969.983.976.976 0 01-.969-.983c0-.542.434-.982.97-.982zm4.844 0c.535 0 .969.44.969.982a.976.976 0 01-.969.983.976.976 0 01-.969-.983c0-.542.434-.982.97-.982z"/>
        </svg>
        <span>{{ t('recharge.qrcodeHint') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'

interface Props {
  codeUrl: string // Native 支付返回的 code_url
  size?: number // 二维码尺寸（像素）
}

const props = withDefaults(defineProps<Props>(), {
  size: 200
})

const emit = defineEmits<{
  (e: 'generated'): void
  (e: 'error', error: Error): void
}>()

const { t } = useI18n()

// 状态
const error = ref(false)
const canvasRef = ref<HTMLCanvasElement | null>(null)
const retryCount = ref(0)
const MAX_RETRIES = 3

// 验证微信支付 URL 格式
const isValidWeChatPayUrl = (url: string): boolean => {
  // 微信支付 Native 支付 URL 格式: weixin://wxpay/bizpayurl?pr=xxx
  return url.startsWith('weixin://wxpay/') || url.startsWith('https://wx.tenpay.com/')
}

// 生成二维码
const generateQRCode = async () => {
  if (!props.codeUrl) {
    return
  }

  // 验证 URL 格式
  if (!isValidWeChatPayUrl(props.codeUrl)) {
    console.warn('[QRCodeDisplay] Invalid WeChat Pay URL format:', props.codeUrl)
    error.value = true
    emit('error', new Error('Invalid WeChat Pay URL format'))
    return
  }

  error.value = false

  // 等待 DOM 更新，确保 canvas 已渲染
  await nextTick()

  if (!canvasRef.value) {
    error.value = true
    emit('error', new Error('Canvas element not found'))
    return
  }

  try {
    await QRCode.toCanvas(canvasRef.value, props.codeUrl, {
      width: props.size,
      margin: 1,
      color: {
        dark: '#000000',
        light: '#ffffff'
      },
      errorCorrectionLevel: 'M'
    })

    emit('generated')
  } catch (err) {
    console.error('[QRCodeDisplay] Failed to generate QR code:', err)
    error.value = true
    emit('error', err instanceof Error ? err : new Error(String(err)))
  }
}

// 重试生成
const retryGenerate = () => {
  if (retryCount.value >= MAX_RETRIES) {
    console.warn('[QRCodeDisplay] Max retry attempts reached')
    return
  }

  retryCount.value++
  error.value = false
  // 需要等待 DOM 更新后再生成
  nextTick(() => {
    generateQRCode()
  })
}

// 重置重试计数（当 codeUrl 变化时）
const resetRetryCount = () => {
  retryCount.value = 0
}

// 监听 codeUrl 变化
watch(
  () => props.codeUrl,
  (newUrl, oldUrl) => {
    if (newUrl && newUrl !== oldUrl) {
      resetRetryCount()
      generateQRCode()
    }
  }
)

// 组件挂载时生成二维码
onMounted(() => {
  if (props.codeUrl) {
    generateQRCode()
  }
})

// 暴露方法
defineExpose({
  regenerate: generateQRCode
})
</script>
