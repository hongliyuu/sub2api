<template>
  <div class="flex min-h-screen items-center justify-center bg-gradient-to-br from-primary-50 to-blue-50 px-4 py-8 dark:from-dark-900 dark:to-dark-800">
    <!-- Loading State -->
    <div v-if="loading" class="flex flex-col items-center gap-4">
      <svg
        class="h-10 w-10 animate-spin text-primary-600 dark:text-primary-400"
        fill="none"
        viewBox="0 0 24 24"
      >
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
      </svg>
      <p class="text-gray-500 dark:text-dark-400">{{ t('lottery.share.loading') }}</p>
    </div>

    <!-- 404 / Not Found State -->
    <div v-else-if="notFound" class="w-full max-w-md text-center">
      <div class="rounded-3xl bg-white p-8 shadow-xl dark:bg-dark-800">
        <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
          <svg class="h-8 w-8 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
          </svg>
        </div>
        <h2 class="mb-2 text-xl font-bold text-gray-900 dark:text-white">{{ t('lottery.share.notFound') }}</h2>
        <p class="mb-6 text-gray-500 dark:text-dark-400">{{ t('lottery.share.notFoundDesc') }}</p>
        <router-link to="/" class="btn btn-primary inline-block">{{ t('lottery.share.backHome') }}</router-link>
      </div>
    </div>

    <!-- Activity Card -->
    <div v-else-if="activity" class="w-full max-w-lg">
      <div class="rounded-3xl bg-white p-8 shadow-xl dark:bg-dark-800">
        <!-- Header -->
        <div class="mb-6 text-center">
          <div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-500 to-primary-600 shadow-lg shadow-primary-500/30">
            <svg class="h-8 w-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 11.25v8.25a1.5 1.5 0 01-1.5 1.5H5.25a1.5 1.5 0 01-1.5-1.5v-8.25M12 4.875A2.625 2.625 0 109.375 7.5H12m0-2.625V7.5m0-2.625A2.625 2.625 0 1114.625 7.5H12m0 0V21m-8.625-9.75h18c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125h-18c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z" />
            </svg>
          </div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ activity.title }}</h1>
          <p v-if="activity.description" class="mt-2 text-gray-500 dark:text-dark-400">{{ activity.description }}</p>
        </div>

        <!-- Activity Info -->
        <div class="mb-6 grid grid-cols-3 gap-4 rounded-2xl bg-gray-50 p-4 dark:bg-dark-700">
          <div class="text-center">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('lottery.share.drawTime') }}</p>
            <p class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ formatDrawTime }}</p>
          </div>
          <div class="text-center">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('lottery.share.participants') }}</p>
            <p class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ activity.participant_count }}</p>
          </div>
          <div class="text-center">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('lottery.share.winRate') }}</p>
            <p class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">{{ (activity.base_win_rate * 100).toFixed(0) }}%</p>
          </div>
        </div>

        <!-- Prize Info -->
        <div class="mb-8 space-y-3">
          <div class="flex items-center gap-3 rounded-xl border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-800/50 dark:bg-yellow-900/20">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-yellow-100 dark:bg-yellow-900/40">
              <svg class="h-5 w-5 text-yellow-600 dark:text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M11.48 3.499a.562.562 0 011.04 0l2.125 5.111a.563.563 0 00.475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 00-.182.557l1.285 5.385a.562.562 0 01-.84.61l-4.725-2.885a.563.563 0 00-.586 0L6.982 20.54a.562.562 0 01-.84-.61l1.285-5.386a.562.562 0 00-.182-.557l-4.204-3.602a.563.563 0 01.321-.988l5.518-.442a.563.563 0 00.475-.345L11.48 3.5z" />
              </svg>
            </div>
            <div>
              <p class="text-sm font-medium text-yellow-800 dark:text-yellow-300">{{ t('lottery.share.winnerPrize') }}</p>
              <p class="text-sm text-yellow-700 dark:text-yellow-400">{{ t('lottery.share.rechargeDiscount', { percent: activity.winner_discount_percent }) }}</p>
            </div>
          </div>

          <div class="flex items-center gap-3 rounded-xl border border-blue-200 bg-blue-50 p-4 dark:border-blue-800/50 dark:bg-blue-900/20">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900/40">
              <svg class="h-5 w-5 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 6v.75m0 3v.75m0 3v.75m0 3V18m-9-5.25h5.25M7.5 15h3M3.375 5.25c-.621 0-1.125.504-1.125 1.125v3.026a2.999 2.999 0 010 5.198v3.026c0 .621.504 1.125 1.125 1.125h17.25c.621 0 1.125-.504 1.125-1.125v-3.026a2.999 2.999 0 010-5.198V6.375c0-.621-.504-1.125-1.125-1.125H3.375z" />
              </svg>
            </div>
            <div>
              <p class="text-sm font-medium text-blue-800 dark:text-blue-300">{{ t('lottery.share.participationPrize') }}</p>
              <p class="text-sm text-blue-700 dark:text-blue-400">{{ t('lottery.share.subscriptionReduction', { amount: activity.loser_coupon_amount }) }}</p>
            </div>
          </div>
        </div>

        <!-- CTA Button -->
        <button
          v-if="isActive"
          @click="handleParticipate"
          class="w-full rounded-2xl bg-gradient-to-r from-primary-500 to-primary-600 px-6 py-4 text-lg font-bold text-white shadow-lg shadow-primary-500/30 transition-all hover:from-primary-600 hover:to-primary-700 hover:shadow-xl hover:shadow-primary-500/40 active:scale-[0.98]"
        >
          {{ t('lottery.share.participate') }}
        </button>
        <button
          v-else
          disabled
          class="w-full cursor-not-allowed rounded-2xl bg-gray-300 px-6 py-4 text-lg font-bold text-gray-500 dark:bg-dark-600 dark:text-dark-400"
        >
          {{ t('lottery.share.activityEnded') }}
        </button>

        <!-- Auth Links -->
        <div class="mt-6 flex items-center justify-center gap-4 text-sm">
          <router-link to="/login" class="text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300">
            {{ t('lottery.share.hasAccount') }}
          </router-link>
          <span class="text-gray-300 dark:text-dark-600">|</span>
          <router-link to="/register" class="text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300">
            {{ t('lottery.share.newUser') }}
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { getByShareCode } from '@/api/lottery'
import type { LotteryActivity } from '@/types'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const activity = ref<LotteryActivity | null>(null)
const loading = ref(true)
const notFound = ref(false)

const isActive = computed(() => {
  if (!activity.value) return false
  return activity.value.status === 'active'
})

const formatDrawTime = computed(() => {
  if (!activity.value?.draw_at) return '-'
  const date = new Date(activity.value.draw_at)
  return date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
})

const handleParticipate = () => {
  if (!activity.value) return
  router.push(`/lottery/${activity.value.id}`)
}

onMounted(async () => {
  const code = route.params.code as string
  if (!code) {
    notFound.value = true
    loading.value = false
    return
  }
  try {
    activity.value = await getByShareCode(code)
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
})
</script>
