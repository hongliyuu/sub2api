<template>
  <section id="how-it-works" class="scroll-mt-16 py-12 sm:py-16">
    <h2 class="mb-3 text-center text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">
      {{ t('home.howItWorks.title') }}
    </h2>
    <p class="mb-8 text-center text-sm text-gray-500 dark:text-dark-400 sm:mb-12 sm:text-base">
      {{ t('home.howItWorks.subtitle') }}
    </p>

    <!-- 3-Step Overview -->
    <div class="mb-8 grid gap-4 sm:grid-cols-3 sm:gap-6">
      <div v-for="(step, i) in steps" :key="i" class="relative rounded-xl border border-gray-200/50 bg-white/60 p-5 text-center backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/60">
        <div class="mx-auto mb-3 flex h-10 w-10 items-center justify-center rounded-full bg-primary-100 text-lg font-bold text-primary-600 dark:bg-primary-900/30 dark:text-primary-400">
          {{ i + 1 }}
        </div>
        <h3 class="mb-1 text-sm font-semibold text-gray-900 dark:text-white">{{ step.title }}</h3>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ step.desc }}</p>
        <!-- Arrow connector (hidden on mobile) -->
        <div v-if="i < 2" class="absolute -right-4 top-1/2 hidden -translate-y-1/2 sm:block">
          <Icon name="arrowRight" size="lg" class="text-primary-400 dark:text-primary-500" />
        </div>
      </div>
    </div>

    <!-- Expandable Detail -->
    <div class="text-center">
      <button
        @click="showDetail = !showDetail"
        class="inline-flex items-center gap-2 rounded-full border border-gray-200 bg-white px-5 py-2 text-sm font-medium text-gray-700 transition-all hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700"
      >
        {{ showDetail ? t('home.howItWorks.hideDetail') : t('home.howItWorks.showDetail') }}
        <svg class="h-4 w-4 transition-transform" :class="showDetail ? 'rotate-180' : ''" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
    </div>

    <div v-if="showDetail" class="mt-6">
      <!-- Tool Tabs -->
      <div class="mb-6 flex justify-center gap-2">
        <button
          v-for="tool in tools"
          :key="tool.id"
          @click="activeTool = tool.id"
          class="flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-all"
          :class="activeTool === tool.id
            ? 'bg-gray-900 text-white shadow-md dark:bg-white dark:text-gray-900'
            : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-dark-300 dark:hover:bg-dark-600'"
        >
          <img :src="tool.logo" :alt="tool.name" class="h-5 w-5 rounded object-contain" />
          {{ tool.name }}
        </button>
      </div>

      <!-- ToolTutorial (compact mode) -->
      <ToolTutorial :tool="activeTool" :compact="true" :video-config="toolVideoConfig" />

      <!-- Link to full guide -->
      <div class="mt-4 text-center">
        <router-link to="/install-guide" class="text-sm font-medium text-primary-600 hover:text-primary-500 dark:text-primary-400">
          {{ t('home.howItWorks.fullGuide') }} →
        </router-link>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ToolTutorial from '@/components/install-guide/ToolTutorial.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const props = defineProps<{
  videoConfig?: Record<string, any> | null
}>()

const showDetail = ref(false)
const activeTool = ref<'claude-code' | 'codex' | 'gemini-cli'>('claude-code')

const steps = computed(() => [
  { title: t('home.howItWorks.step1.title'), desc: t('home.howItWorks.step1.desc') },
  { title: t('home.howItWorks.step2.title'), desc: t('home.howItWorks.step2.desc') },
  { title: t('home.howItWorks.step3.title'), desc: t('home.howItWorks.step3.desc') },
])

const tools = [
  { id: 'claude-code' as const, name: 'Claude Code', logo: '/llmLogo/Claude.png' },
  { id: 'codex' as const, name: 'Codex CLI', logo: '/llmLogo/ChatGPT.png' },
  { id: 'gemini-cli' as const, name: 'Gemini CLI', logo: '/llmLogo/Gemini.jpg' },
]

const toolVideoConfig = computed(() => {
  if (!props.videoConfig) return null
  const key = activeTool.value === 'claude-code' ? 'claude_code' : activeTool.value === 'gemini-cli' ? 'gemini_cli' : 'codex'
  return props.videoConfig[key] || null
})
</script>
