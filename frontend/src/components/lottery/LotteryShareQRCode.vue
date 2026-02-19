<template>
  <div class="flex flex-col items-center gap-3">
    <canvas ref="canvasRef" />
    <div class="flex gap-2">
      <button @click="copyLink" class="btn btn-secondary btn-sm">
        {{ t('lottery.share.copyLink') }}
      </button>
    </div>
    <p v-if="copied" class="text-sm text-green-600">{{ t('lottery.share.linkCopied') }}</p>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import { useClipboard } from '@/composables/useClipboard'

const { t } = useI18n()
const { copied, copyToClipboard } = useClipboard()

const props = withDefaults(defineProps<{
  shareCode: string
  size?: number
}>(), {
  size: 200
})

const canvasRef = ref<HTMLCanvasElement>()
const shareUrl = computed(() => `${window.location.origin}/lottery/share/${props.shareCode}`)

async function generateQR() {
  if (!canvasRef.value) return
  await QRCode.toCanvas(canvasRef.value, shareUrl.value, {
    width: props.size,
    margin: 2,
    color: {
      dark: '#000000',
      light: '#ffffff'
    }
  })
}

function copyLink() {
  copyToClipboard(shareUrl.value, t('lottery.share.linkCopied'))
}

onMounted(generateQR)
watch(() => props.shareCode, generateQR)
</script>
