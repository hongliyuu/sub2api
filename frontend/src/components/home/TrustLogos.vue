<template>
  <section id="trust-logos" class="py-8 sm:py-12">
    <p class="mb-6 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('home.trustLogos.subtitle') }}</p>
    <div class="flex flex-wrap items-center justify-center gap-6 sm:gap-10">
      <router-link
        v-for="logo in logos"
        :key="logo.name"
        :to="logo.link"
        class="group flex items-center gap-2 transition-all"
      >
        <img
          :src="logo.src"
          :alt="logo.name"
          class="h-8 w-8 rounded-lg object-contain transition-all sm:h-10 sm:w-10"
        />
        <span class="text-sm font-medium text-gray-600 transition-colors group-hover:text-gray-900 dark:text-dark-300 dark:group-hover:text-white">
          {{ logo.name }}
        </span>
      </router-link>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const logos = [
  { name: 'Claude', src: '/llmLogo/Claude.png', link: '/install-guide#claude-code' },
  { name: 'GPT', src: '/llmLogo/ChatGPT.png', link: '/install-guide#codex' },
  { name: 'Gemini', src: '/llmLogo/Gemini.jpg', link: '/install-guide#gemini' },
  { name: 'Antigravity', src: '/llmLogo/Antigravity.jpg', link: '#' },
]
</script>
