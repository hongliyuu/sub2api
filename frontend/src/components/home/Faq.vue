<template>
  <section id="faq" class="scroll-mt-16 py-12 sm:py-16">
    <h2 class="mb-3 text-center text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">
      {{ t('home.faq.title') }}
    </h2>
    <p class="mb-8 text-center text-sm text-gray-500 dark:text-dark-400 sm:mb-12 sm:text-base">
      {{ t('home.faq.subtitle') }}
    </p>

    <div class="space-y-3">
      <div
        v-for="(item, i) in faqItems"
        :key="i"
        class="rounded-xl border border-gray-200/50 bg-white/60 backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/60"
      >
        <button
          @click="toggle(i)"
          class="flex w-full items-center justify-between px-5 py-4 text-left"
        >
          <span class="text-sm font-medium text-gray-900 dark:text-white">{{ item.q }}</span>
          <svg
            class="h-4 w-4 shrink-0 text-gray-400 transition-transform"
            :class="openIndex === i ? 'rotate-180' : ''"
            fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
          </svg>
        </button>
        <div v-if="openIndex === i" class="border-t border-gray-100 px-5 py-4 dark:border-dark-700">
          <p class="text-sm leading-relaxed text-gray-600 dark:text-dark-300">{{ item.a }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const openIndex = ref<number | null>(null)

function toggle(i: number) {
  openIndex.value = openIndex.value === i ? null : i
}

const faqItems = computed(() => [
  { q: t('home.faq.items.apiKey.q'), a: t('home.faq.items.apiKey.a') },
  { q: t('home.faq.items.models.q'), a: t('home.faq.items.models.a') },
  { q: t('home.faq.items.billing.q'), a: t('home.faq.items.billing.a') },
  { q: t('home.faq.items.support.q'), a: t('home.faq.items.support.a') },
  { q: t('home.faq.items.refund.q'), a: t('home.faq.items.refund.a') },
])
</script>
