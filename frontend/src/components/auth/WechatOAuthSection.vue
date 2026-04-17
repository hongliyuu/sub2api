<template>
  <div class="space-y-4">
    <div class="space-y-2">
      <button type="button" :disabled="disabled" class="btn btn-secondary w-full" @click="startLogin">
        <span
          class="mr-2 inline-flex h-5 w-5 items-center justify-center rounded-full bg-emerald-100 text-xs font-semibold text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300"
        >
          W
        </span>
        {{ t('auth.wechat.signIn') }}
      </button>
      <p class="text-xs text-gray-500 dark:text-dark-400">
        {{ loginHint }}
      </p>
    </div>

    <div v-if="showDivider" class="flex items-center gap-3">
      <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
      <span class="text-xs text-gray-500 dark:text-dark-400">
        {{ t('auth.oauthOrContinue') }}
      </span>
      <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'

withDefaults(defineProps<{
  disabled?: boolean
  showDivider?: boolean
}>(), {
  showDivider: true
})

const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()

const userAgent = typeof navigator !== 'undefined' ? navigator.userAgent : ''
const isWeChatBrowser = /MicroMessenger/i.test(userAgent)
const isMobileBrowser = /Android|iPhone|iPad|iPod|Mobile/i.test(userAgent)
const isDesktopBrowser = !isMobileBrowser

const loginHint = computed(() =>
  isWeChatBrowser ? t('auth.wechat.inAppHint') : t('auth.wechat.externalHint')
)

function startLogin(): void {
  if (!isWeChatBrowser && isDesktopBrowser) {
    appStore.showInfo(t('auth.wechat.desktopHint'), 3200)
  } else if (!isWeChatBrowser) {
    appStore.showInfo(t('auth.wechat.mobileBrowserHint'), 3200)
  }
  const redirectTo = (route.query.redirect as string) || '/dashboard'
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api/v1'
  const normalized = apiBase.replace(/\/$/, '')
  const startURL = `${normalized}/auth/oauth/wechat/start?redirect=${encodeURIComponent(redirectTo)}`
  window.location.href = startURL
}
</script>
