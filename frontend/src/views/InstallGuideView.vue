<template>
  <div
    class="relative flex min-h-screen flex-col overflow-hidden bg-gradient-to-br from-gray-50 via-primary-50/30 to-gray-100 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950"
  >
    <!-- Background Decorations -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div
        class="absolute -right-40 -top-40 h-96 w-96 rounded-full bg-primary-400/20 blur-3xl"
      ></div>
      <div
        class="absolute -bottom-40 -left-40 h-96 w-96 rounded-full bg-primary-500/15 blur-3xl"
      ></div>
      <div
        class="absolute left-1/3 top-1/4 h-72 w-72 rounded-full bg-primary-300/10 blur-3xl"
      ></div>
      <div
        class="absolute bottom-1/4 right-1/4 h-64 w-64 rounded-full bg-primary-400/10 blur-3xl"
      ></div>
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(217,119,87,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(217,119,87,0.03)_1px,transparent_1px)] bg-[size:64px_64px]"
      ></div>
    </div>

    <!-- Header -->
    <header class="relative z-20 px-6 py-4">
      <nav class="mx-auto flex max-w-6xl items-center justify-between">
        <!-- Logo -->
        <router-link to="/home" class="flex items-center gap-3">
          <div class="h-10 w-10 overflow-hidden rounded-xl shadow-md">
            <img :src="currentLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="text-lg font-semibold text-gray-900 dark:text-white">{{ siteName }}</span>
        </router-link>

        <!-- Nav Actions -->
        <div class="flex items-center gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t(`nav.${themeMode === 'light' ? 'darkMode' : themeMode === 'dark' ? 'autoMode' : 'lightMode'}`)"
          >
            <Icon v-if="themeMode === 'light'" name="sun" size="md" class="text-amber-500" />
            <Icon v-else-if="themeMode === 'dark'" name="moon" size="md" class="text-indigo-400" />
            <Icon v-else name="clock" size="md" class="text-emerald-500" />
          </button>

          <!-- Back to Home -->
          <router-link
            to="/home"
            class="inline-flex items-center rounded-full bg-gray-900 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            {{ t('common.back') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-4 py-8 sm:px-6 sm:py-16">
      <div class="mx-auto max-w-6xl">
        <!-- Page Title -->
        <div class="mb-8 text-center sm:mb-12">
          <h1 class="mb-3 text-3xl font-bold text-gray-900 dark:text-white sm:mb-4 sm:text-4xl lg:text-5xl">
            {{ t('installGuide.title') }}
          </h1>
          <p class="mx-auto max-w-2xl text-base text-gray-600 dark:text-dark-300 sm:text-lg">
            {{ t('installGuide.subtitle') }}
          </p>
        </div>

        <!-- Tool Tabs -->
        <div class="mb-8 flex justify-center">
          <div class="inline-flex rounded-xl bg-gray-100 p-1 dark:bg-dark-800">
            <button
              v-for="tool in tools"
              :key="tool.id"
              @click="activeTool = tool.id"
              class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all sm:px-6 sm:py-2.5"
              :class="activeTool === tool.id
                ? 'bg-white text-gray-900 shadow-md dark:bg-dark-700 dark:text-white'
                : 'text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-200'"
            >
              <img :src="tool.logo" :alt="tool.name" class="h-5 w-5 rounded object-contain" />
              <span class="hidden sm:inline">{{ tool.name }}</span>
            </button>
          </div>
        </div>

        <!-- Tool Tutorial -->
        <div class="card overflow-hidden">
          <div class="card-body">
            <ToolTutorial
              :key="activeTool"
              :tool="activeTool"
              :video-config="currentVideoConfig"
            />
          </div>
        </div>

        <!-- Tips Section -->
        <div class="mt-8 sm:mt-12">
          <div class="rounded-2xl border border-primary-200/50 bg-gradient-to-br from-primary-50 to-primary-100/50 p-4 dark:border-primary-800/50 dark:from-primary-900/20 dark:to-primary-800/10 sm:p-6">
            <div class="flex items-start gap-3 sm:gap-4">
              <div class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-primary-500 text-white sm:h-10 sm:w-10">
                <Icon name="infoCircle" size="md" />
              </div>
              <div>
                <h3 class="mb-2 text-base font-semibold text-gray-900 dark:text-white sm:text-lg">
                  {{ t('installGuide.tips.title') }}
                </h3>
                <ul class="space-y-1.5 text-xs text-gray-600 dark:text-dark-300 sm:space-y-2 sm:text-sm">
                  <li class="flex items-start gap-2">
                    <span class="mt-1.5 h-1 w-1 flex-shrink-0 rounded-full bg-primary-500 sm:mt-2"></span>
                    {{ t('installGuide.tips.tip1') }}
                  </li>
                  <li class="flex items-start gap-2">
                    <span class="mt-1.5 h-1 w-1 flex-shrink-0 rounded-full bg-primary-500 sm:mt-2"></span>
                    {{ t('installGuide.tips.tip2') }}
                  </li>
                  <li class="flex items-start gap-2">
                    <span class="mt-1.5 h-1 w-1 flex-shrink-0 rounded-full bg-primary-500 sm:mt-2"></span>
                    {{ t('installGuide.tips.tip3') }}
                  </li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-gray-200/50 px-6 py-8 dark:border-dark-800/50">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores'
import { useTheme } from '@/composables/useTheme'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import ToolTutorial from '@/components/install-guide/ToolTutorial.vue'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const { isDark, themeMode, toggleTheme } = useTheme()

// Tool definitions
const tools = [
  { id: 'claude-code' as const, name: 'Claude Code', logo: '/llmLogo/Claude.png' },
  { id: 'codex' as const, name: 'Codex CLI', logo: '/llmLogo/ChatGPT.png' },
  { id: 'gemini-cli' as const, name: 'Gemini CLI', logo: '/llmLogo/Gemini.jpg' }
]

const activeTool = ref<'claude-code' | 'codex' | 'gemini-cli'>('claude-code')

// Handle hash scroll on mount and set active tool from hash
onMounted(() => {
  nextTick(() => {
    const hash = route.hash?.replace('#', '')
    if (hash === 'codex') activeTool.value = 'codex'
    else if (hash === 'gemini' || hash === 'gemini-cli') activeTool.value = 'gemini-cli'
  })
})

// Site settings
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Code80')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteLogoDark = computed(() => appStore.cachedPublicSettings?.site_logo_dark || appStore.siteLogoDark || '')

const currentLogo = computed(() => {
  if (isDark.value && siteLogoDark.value) {
    return siteLogoDark.value
  }
  return siteLogo.value
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Video config from settings
const videoConfig = computed(() => {
  const raw = appStore.cachedPublicSettings?.install_guide_videos || ''
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
})

const toolKeyMap: Record<string, string> = {
  'claude-code': 'claude_code',
  codex: 'codex',
  'gemini-cli': 'gemini_cli'
}

const currentVideoConfig = computed(() => {
  if (!videoConfig.value) return null
  const key = toolKeyMap[activeTool.value]
  return videoConfig.value[key] || null
})
</script>
