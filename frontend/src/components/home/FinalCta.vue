<template>
  <section class="py-12 sm:py-16">
    <div class="rounded-2xl bg-gradient-to-br from-primary-500 to-primary-700 p-8 text-center shadow-xl sm:p-12">
      <h2 class="mb-3 text-2xl font-bold text-white sm:text-3xl">
        {{ t('home.finalCta.title') }}
      </h2>
      <p class="mx-auto mb-6 max-w-lg text-sm text-primary-100 sm:text-base">
        {{ t('home.finalCta.subtitle') }}
      </p>
      <div class="flex flex-wrap items-center justify-center gap-3">
        <router-link
          :to="isAuthenticated ? dashboardPath : '/register'"
          class="rounded-full bg-white px-6 py-2.5 text-sm font-semibold text-primary-600 shadow-lg transition-all hover:bg-gray-50 hover:shadow-xl"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.finalCta.register') }}
        </router-link>
        <router-link
          to="/install-guide"
          class="rounded-full border border-white/30 px-6 py-2.5 text-sm font-semibold text-white transition-all hover:bg-white/10"
        >
          {{ t('home.installGuide') }}
        </router-link>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores'

const { t } = useI18n()

const authStore = useAuthStore()

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
</script>
