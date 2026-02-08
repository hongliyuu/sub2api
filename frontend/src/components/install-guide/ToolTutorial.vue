<template>
  <div class="space-y-4 sm:space-y-6">
    <!-- Tool Header -->
    <div class="flex items-start gap-3 sm:gap-4">
      <img
        :src="toolConfig.logo"
        :alt="toolConfig.name"
        class="h-10 w-10 rounded-xl shadow-md sm:h-12 sm:w-12"
      />
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white sm:text-lg">
            {{ toolConfig.name }}
          </h2>
          <a
            :href="toolConfig.docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-1 rounded-md bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-200 dark:bg-dark-700 dark:text-dark-300 dark:hover:bg-dark-600"
          >
            <Icon name="externalLink" size="xs" />
            {{ t('installGuide.officialDocs') }}
          </a>
        </div>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400 sm:text-sm">
          {{ toolConfig.description }}
        </p>
      </div>
    </div>

    <!-- Compact mode: collapsible wrapper -->
    <template v-if="compact">
      <button
        @click="expanded = !expanded"
        class="flex w-full items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm font-medium text-gray-700 transition-colors hover:bg-gray-100 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300 dark:hover:bg-dark-700"
      >
        <Icon :name="expanded ? 'chevronDown' : 'chevronRight'" size="sm" />
        {{ expanded ? t('installGuide.verifyInstall') : t('installGuide.prerequisites') }}
        <span class="text-xs text-gray-400">
          {{ expanded ? '' : '...' }}
        </span>
      </button>
    </template>

    <!-- Tutorial Steps -->
    <div
      v-show="!compact || expanded"
      class="space-y-4 sm:space-y-6"
    >
      <!-- OS Selector -->
      <OsSelector v-model="selectedOs" :color="toolConfig.color" />

      <!-- Step 1: Prerequisites -->
      <TutorialStep :step="1" :title="t('installGuide.prerequisites')" :color="toolConfig.color">
        <div class="space-y-2 pl-7 sm:pl-8">
          <p class="text-xs text-gray-600 dark:text-dark-400 sm:text-sm">
            Node.js {{ toolConfig.node }} ({{ t('installGuide.recommended') }})
          </p>
          <CodeBlock code="node --version" language="bash" />
          <template v-if="tool === 'codex' && selectedOs === 'linux'">
            <p class="text-xs text-gray-500 dark:text-dark-400 sm:text-sm">
              {{ t('installGuide.codex.linuxSandbox') }}
            </p>
            <CodeBlock code="sudo apt-get install bubblewrap" language="bash" />
          </template>
        </div>
      </TutorialStep>

      <!-- Step 2: Install -->
      <TutorialStep :step="2" :title="t('installGuide.installCommand')" :color="toolConfig.color">
        <div class="pl-7 sm:pl-8">
          <CodeBlock :code="toolConfig.installCmd" language="bash" />
        </div>
      </TutorialStep>

      <!-- Step 3: Configure API Key -->
      <TutorialStep :step="3" :title="t('installGuide.configApiKey')" :color="toolConfig.color">
        <div class="space-y-2 pl-7 sm:pl-8">
          <CodeBlock :code="configCommand" language="bash" />
        </div>
      </TutorialStep>

      <!-- Step 4: Verify -->
      <TutorialStep :step="4" :title="t('installGuide.verifyInstall')" :color="toolConfig.color">
        <div class="pl-7 sm:pl-8">
          <CodeBlock :code="toolConfig.verifyCmd" language="bash" />
        </div>
      </TutorialStep>

      <!-- Video Embed -->
      <VideoEmbed
        v-if="videoConfig?.overview"
        :url="videoConfig.overview"
        :title="toolConfig.name"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import Icon from '@/components/icons/Icon.vue'
import CodeBlock from '@/components/common/CodeBlock.vue'
import OsSelector from './OsSelector.vue'
import TutorialStep from './TutorialStep.vue'
import VideoEmbed from './VideoEmbed.vue'

const props = withDefaults(defineProps<{
  tool: 'claude-code' | 'codex' | 'gemini-cli'
  compact?: boolean
  videoConfig?: { overview?: string } | null
}>(), {
  compact: false,
  videoConfig: null
})

const { t } = useI18n()
const appStore = useAppStore()

const selectedOs = ref('macos')
const expanded = ref(false)

const apiBaseUrl = computed(() =>
  appStore.cachedPublicSettings?.api_base_url || window.location.origin
)

interface ToolConfig {
  name: string
  color: string
  logo: string
  gradient: string
  docUrl: string
  node: string
  envVar: string
  baseUrlEnvVar: string
  baseUrlValue: string
  installCmd: string
  verifyCmd: string
  description: string
}

const toolConfig = computed<ToolConfig>(() => {
  const configs: Record<string, ToolConfig> = {
    'claude-code': {
      name: 'Claude Code',
      color: 'primary',
      logo: '/llmLogo/Claude.png',
      gradient: 'from-[#d97757] to-[#c45a3a]',
      docUrl: 'https://docs.anthropic.com/en/docs/claude-code',
      node: '18+',
      envVar: 'ANTHROPIC_API_KEY',
      baseUrlEnvVar: 'ANTHROPIC_BASE_URL',
      baseUrlValue: apiBaseUrl.value,
      installCmd: 'npm install -g @anthropic-ai/claude-code',
      verifyCmd: 'claude --version',
      description: t('installGuide.claudeCode.description')
    },
    codex: {
      name: 'Codex CLI',
      color: 'emerald',
      logo: '/llmLogo/ChatGPT.png',
      gradient: 'from-[#10a37f] to-[#0d8a6a]',
      docUrl: 'https://github.com/openai/codex',
      node: '22+',
      envVar: 'OPENAI_API_KEY',
      baseUrlEnvVar: 'OPENAI_BASE_URL',
      baseUrlValue: `${apiBaseUrl.value}/v1`,
      installCmd: 'npm install -g @openai/codex',
      verifyCmd: 'codex --version',
      description: t('installGuide.codex.description')
    },
    'gemini-cli': {
      name: 'Gemini CLI',
      color: 'blue',
      logo: '/llmLogo/Gemini.jpg',
      gradient: 'from-[#4285f4] to-[#1a73e8]',
      docUrl: 'https://github.com/google-gemini/gemini-cli',
      node: '20+',
      envVar: 'GEMINI_API_KEY',
      baseUrlEnvVar: 'GEMINI_API_BASE_URL',
      baseUrlValue: `${apiBaseUrl.value}/v1beta`,
      installCmd: 'npm install -g @google/gemini-cli',
      verifyCmd: 'gemini --version',
      description: t('installGuide.gemini.description')
    }
  }
  return configs[props.tool]
})

const configCommand = computed(() => {
  const cfg = toolConfig.value
  if (selectedOs.value === 'windows') {
    return `$env:${cfg.envVar}="your-api-key"\n$env:${cfg.baseUrlEnvVar}="${cfg.baseUrlValue}"`
  }
  return `export ${cfg.envVar}="your-api-key"\nexport ${cfg.baseUrlEnvVar}="${cfg.baseUrlValue}"`
})
</script>
