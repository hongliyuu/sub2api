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
        {{ expanded ? t('installGuide.step1.title') : t('installGuide.step1.title') }}
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

      <!-- Step 1: Install Node.js -->
      <TutorialStep :step="1" :title="t('installGuide.step1.title')" :color="toolConfig.color">
        <div class="space-y-3 pl-7 sm:pl-8">
          <p class="text-xs text-gray-600 dark:text-dark-400 sm:text-sm">
            {{ t('installGuide.step1.nodeRequired', { version: toolConfig.nodeVersion }) }}
          </p>

          <!-- macOS -->
          <template v-if="selectedOs === 'macos'">
            <p class="text-xs font-medium text-gray-700 dark:text-dark-300 sm:text-sm">
              {{ t('installGuide.step1.recommendedMethod') }}
            </p>
            <CodeBlock code="brew install node" language="bash" />
            <details class="group">
              <summary class="cursor-pointer text-xs text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-300 sm:text-sm">
                {{ t('installGuide.step1.otherMethods') }}
              </summary>
              <div class="mt-2 space-y-2">
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('installGuide.step1.brewNote') }}
                </p>
                <CodeBlock code='/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"' language="bash" />
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('installGuide.step1.downloadFromNodejs') }}
                </p>
              </div>
            </details>
          </template>

          <!-- Windows -->
          <template v-else-if="selectedOs === 'windows'">
            <p class="text-xs font-medium text-gray-700 dark:text-dark-300 sm:text-sm">
              {{ t('installGuide.step1.recommendedMethod') }}
            </p>
            <CodeBlock code="winget install OpenJS.NodeJS.LTS" language="bash" />
            <details class="group">
              <summary class="cursor-pointer text-xs text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-300 sm:text-sm">
                {{ t('installGuide.step1.otherMethods') }}
              </summary>
              <div class="mt-2 space-y-2">
                <CodeBlock code="choco install nodejs-lts" language="bash" />
                <CodeBlock code="scoop install nodejs-lts" language="bash" />
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('installGuide.step1.downloadFromNodejs') }}
                </p>
              </div>
            </details>
          </template>

          <!-- Linux -->
          <template v-else>
            <p class="text-xs font-medium text-gray-700 dark:text-dark-300 sm:text-sm">
              {{ t('installGuide.step1.recommendedMethod') }}
            </p>
            <p class="text-xs text-gray-500 dark:text-dark-400">
              {{ t('installGuide.step1.nodeSourceNote') }}
            </p>
            <CodeBlock :code="`curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -\nsudo apt-get install -y nodejs`" language="bash" />
            <details class="group">
              <summary class="cursor-pointer text-xs text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-300 sm:text-sm">
                {{ t('installGuide.step1.otherMethods') }}
              </summary>
              <div class="mt-2 space-y-2">
                <CodeBlock code="sudo dnf install nodejs" language="bash" />
                <p class="text-xs text-gray-500 dark:text-dark-400">
                  {{ t('installGuide.step1.downloadFromNodejs') }}
                </p>
              </div>
            </details>
          </template>

          <!-- Verify -->
          <p class="text-xs font-medium text-gray-700 dark:text-dark-300 sm:text-sm">
            {{ t('installGuide.step1.verifyNode') }}
          </p>
          <CodeBlock code="node --version" language="bash" />

          <!-- Codex Linux sandbox -->
          <template v-if="tool === 'codex' && selectedOs === 'linux'">
            <p class="text-xs text-gray-500 dark:text-dark-400 sm:text-sm">
              {{ t('installGuide.codex.linuxSandbox') }}
            </p>
            <CodeBlock code="sudo apt-get install bubblewrap" language="bash" />
          </template>
        </div>
      </TutorialStep>

      <!-- Step 2: Install CLI Tool -->
      <TutorialStep :step="2" :title="t('installGuide.step2.title')" :color="toolConfig.color">
        <div class="space-y-3 pl-7 sm:pl-8">
          <CodeBlock :code="toolConfig.installCmd" language="bash" />
          <p class="text-xs text-gray-700 dark:text-dark-300 sm:text-sm">
            {{ t('installGuide.step2.verify') }}
          </p>
          <CodeBlock :code="toolConfig.verifyCmd" language="bash" />
          <p class="text-xs text-gray-500 dark:text-dark-400">
            {{ t('installGuide.step2.adminNote') }}
          </p>
        </div>
      </TutorialStep>

      <!-- Step 3: Configure API -->
      <TutorialStep :step="3" :title="t('installGuide.step3.title')" :color="toolConfig.color">
        <div class="space-y-4 pl-7 sm:pl-8">
          <!-- Get Token -->
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800 sm:p-4">
            <p class="mb-2 text-xs font-medium text-gray-700 dark:text-dark-300 sm:text-sm">
              {{ t('installGuide.step3.getToken') }}
            </p>
            <p class="mb-3 text-xs text-gray-500 dark:text-dark-400">
              {{ t('installGuide.step3.getTokenDesc') }}
            </p>
            <router-link
              to="/keys"
              class="inline-flex items-center gap-1.5 rounded-lg bg-gray-900 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-dark-600 dark:hover:bg-dark-500 sm:text-sm"
            >
              {{ t('installGuide.step3.goToTokenPage') }}
              <Icon name="externalLink" size="xs" />
            </router-link>
            <p class="mt-2 text-xs text-gray-400 dark:text-dark-500">
              {{ t('installGuide.step3.selectGroupHint') }}
            </p>
          </div>

          <!-- Claude Code Config -->
          <template v-if="tool === 'claude-code'">
            <ConfigFileBlock
              :filename="claudeConfigPath"
              :code="claudeConfigContent"
              language="json"
            />
          </template>

          <!-- Codex CLI Config -->
          <template v-else-if="tool === 'codex'">
            <p class="text-xs font-medium text-gray-700 dark:text-dark-300 sm:text-sm">
              {{ t('installGuide.step3.codex.createDir') }}
            </p>
            <CodeBlock :code="codexMkdirCmd" language="bash" />

            <ConfigFileBlock
              :filename="codexConfigPath"
              :code="codexConfigContent"
              language="toml"
            />

            <ConfigFileBlock
              :filename="codexAuthPath"
              :code="codexAuthContent"
              language="json"
            />
          </template>

          <!-- Gemini CLI Config -->
          <template v-else-if="tool === 'gemini-cli'">
            <ConfigFileBlock
              :filename="geminiEnvPath"
              :code="geminiEnvContent"
              language="bash"
            />

            <ConfigFileBlock
              :filename="geminiSettingsPath"
              :code="geminiSettingsContent"
              language="json"
            />
          </template>

          <!-- Replace hint -->
          <p class="text-xs text-amber-600 dark:text-amber-400">
            {{ t('installGuide.step3.replaceKeyHint') }}
          </p>
          <p class="text-xs text-gray-500 dark:text-dark-400">
            {{ t('installGuide.step3.configNote') }}
          </p>
        </div>
      </TutorialStep>

      <!-- Step 4: Start Using -->
      <TutorialStep :step="4" :title="t('installGuide.step4.title')" :color="toolConfig.color">
        <div class="space-y-3 pl-7 sm:pl-8">
          <p class="text-xs text-gray-700 dark:text-dark-300 sm:text-sm">
            {{ t('installGuide.step4.enterProject') }}
          </p>
          <CodeBlock code="cd your-project" language="bash" />

          <p class="text-xs text-gray-700 dark:text-dark-300 sm:text-sm">
            {{ t('installGuide.step4.startTool') }}
          </p>
          <CodeBlock :code="toolConfig.startCmd" language="bash" />

          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-800 sm:p-4">
            <p class="mb-1 text-xs font-medium text-gray-700 dark:text-dark-300 sm:text-sm">
              {{ t('installGuide.step4.firstTimeNotes') }}
            </p>
            <p class="text-xs text-gray-500 dark:text-dark-400">
              {{ firstTimeHint }}
            </p>
          </div>
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
import ConfigFileBlock from './ConfigFileBlock.vue'
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

const siteName = computed(() =>
  appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Code80'
)

interface ToolConfig {
  name: string
  color: string
  logo: string
  docUrl: string
  nodeVersion: string
  installCmd: string
  verifyCmd: string
  startCmd: string
  description: string
}

const toolConfig = computed<ToolConfig>(() => {
  const configs: Record<string, ToolConfig> = {
    'claude-code': {
      name: 'Claude Code',
      color: 'primary',
      logo: '/llmLogo/Claude.png',
      docUrl: 'https://docs.anthropic.com/en/docs/claude-code',
      nodeVersion: '18+',
      installCmd: 'npm install -g @anthropic-ai/claude-code',
      verifyCmd: 'claude --version',
      startCmd: 'claude',
      description: t('installGuide.claudeCode.description')
    },
    codex: {
      name: 'Codex CLI',
      color: 'emerald',
      logo: '/llmLogo/ChatGPT.png',
      docUrl: 'https://github.com/openai/codex',
      nodeVersion: '22+',
      installCmd: 'npm install -g @openai/codex',
      verifyCmd: 'codex --version',
      startCmd: 'codex',
      description: t('installGuide.codex.description')
    },
    'gemini-cli': {
      name: 'Gemini CLI',
      color: 'blue',
      logo: '/llmLogo/Gemini.jpg',
      docUrl: 'https://github.com/google-gemini/gemini-cli',
      nodeVersion: '20+',
      installCmd: 'npm install -g @google/gemini-cli',
      verifyCmd: 'gemini --version',
      startCmd: 'gemini',
      description: t('installGuide.gemini.description')
    }
  }
  return configs[props.tool]
})

// --- Claude Code config ---
const claudeConfigPath = computed(() =>
  selectedOs.value === 'windows'
    ? '%USERPROFILE%\\.claude\\settings.json'
    : '~/.claude/settings.json'
)

const claudeConfigContent = computed(() => {
  return JSON.stringify({
    env: {
      ANTHROPIC_AUTH_TOKEN: 'your-api-key',
      ANTHROPIC_BASE_URL: apiBaseUrl.value
    }
  }, null, 2)
})

// --- Codex CLI config ---
const isWindows = computed(() => selectedOs.value === 'windows')
const codexDir = computed(() => isWindows.value ? '%USERPROFILE%\\.codex' : '~/.codex')

const codexMkdirCmd = computed(() =>
  isWindows.value ? `mkdir %USERPROFILE%\\.codex` : 'mkdir -p ~/.codex'
)

const codexConfigPath = computed(() => `${codexDir.value}/config.toml`)
const codexAuthPath = computed(() => `${codexDir.value}/auth.json`)

const codexConfigContent = computed(() => {
  const providerName = siteName.value.toLowerCase().replace(/\s+/g, '')
  return `model_provider = "${providerName}"
model = "gpt-5.3-codex"
model_reasoning_effort = "high"
network_access = "enabled"
disable_response_storage = true
windows_wsl_setup_acknowledged = true
model_verbosity = "high"

[model_providers.${providerName}]
name = "${providerName}"
base_url = "${apiBaseUrl.value}/v1"
wire_api = "responses"
requires_openai_auth = true`
})

const codexAuthContent = computed(() => {
  return JSON.stringify({
    OPENAI_API_KEY: 'your-api-key'
  }, null, 2)
})

// --- Gemini CLI config ---
const geminiDir = computed(() =>
  isWindows.value ? '%USERPROFILE%\\.gemini' : '~/.gemini'
)

const geminiEnvPath = computed(() => `${geminiDir.value}/.env`)
const geminiSettingsPath = computed(() => `${geminiDir.value}/settings.json`)

const geminiEnvContent = computed(() => {
  return `GOOGLE_GEMINI_BASE_URL=${apiBaseUrl.value}/v1beta
GEMINI_API_KEY=your-api-key
GEMINI_MODEL=gemini-2.5-pro`
})

const geminiSettingsContent = computed(() => {
  return JSON.stringify({
    theme: 'system'
  }, null, 2)
})

// --- First time hints ---
const firstTimeHint = computed(() => {
  switch (props.tool) {
    case 'claude-code':
      return t('installGuide.step4.claudeFirstTime')
    case 'codex':
      return t('installGuide.step4.codexFirstTime')
    case 'gemini-cli':
      return t('installGuide.step4.geminiFirstTime')
    default:
      return ''
  }
})
</script>
