<template>
  <AppLayout>
    <div class="mx-auto max-w-2xl space-y-6">
      <!-- Back Button -->
      <button
        class="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-300"
        @click="router.push('/lottery')"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M15 19l-7-7 7-7" />
        </svg>
        {{ t('lottery.title') }}
      </button>

      <!-- Loading State -->
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <template v-else-if="activity">
        <!-- Activity Header Card -->
        <div class="card overflow-hidden">
          <div class="bg-gradient-to-r from-primary-500 to-primary-600 px-6 py-8 text-center text-white">
            <h1 class="text-2xl font-bold">{{ activity.title }}</h1>
            <span
              :class="['mt-2 inline-block rounded-full px-3 py-1 text-xs font-medium', statusClass]"
            >
              {{ t(`admin.lottery.status.${activity.status}`) }}
            </span>
          </div>

          <div class="p-6 space-y-4">
            <!-- Description -->
            <p v-if="activity.description" class="text-sm text-gray-600 dark:text-dark-400">
              {{ activity.description }}
            </p>

            <!-- Activity Info -->
            <div class="text-sm">
              <div>
                <span class="text-gray-500 dark:text-dark-400">{{ t('lottery.drawTime') }}</span>
                <p class="font-medium text-gray-900 dark:text-white">{{ formatDateTime(activity.draw_at) }}</p>
              </div>
            </div>

            <!-- Prize Info (only show validity-based prize) -->
            <div v-if="validityDays > 0" class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
              <h3 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">{{ t('lottery.share.prizes') }}</h3>
              <div class="flex items-center gap-2 text-sm">
                <span class="inline-block h-2 w-2 rounded-full bg-green-500"></span>
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ t('lottery.detail.prizeDesc', { days: validityDays }) }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Rules -->
        <div class="card p-6">
          <h2 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">{{ t('lottery.detail.rules') }}</h2>
          <ul class="space-y-2 text-sm text-gray-600 dark:text-dark-400">
            <li class="flex items-start gap-2">
              <span class="mt-1 h-1.5 w-1.5 flex-shrink-0 rounded-full bg-primary-500"></span>
              {{ t('lottery.detail.rule1') }}
            </li>
            <li class="flex items-start gap-2">
              <span class="mt-1 h-1.5 w-1.5 flex-shrink-0 rounded-full bg-primary-500"></span>
              {{ t('lottery.detail.rule2') }}
            </li>
          </ul>
        </div>

        <!-- Share (only show if not participated yet) -->
        <div v-if="activity.status === 'active' && activity.share_code && !myParticipation" class="card p-6">
          <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('lottery.share.qrCode') }}</h2>
          <div class="flex flex-col items-center">
            <LotteryShareQRCode :share-code="activity.share_code" :size="180" />
          </div>
        </div>

        <!-- My Status / Participate -->
        <div class="card p-6">
          <h2 class="mb-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('lottery.detail.status') }}</h2>

          <!-- Not participated yet -->
          <div v-if="!myParticipation" class="text-center">
            <p class="mb-4 text-sm text-gray-500 dark:text-dark-400">{{ t('lottery.detail.notParticipated') }}</p>
            <button
              v-if="activity.status === 'active'"
              class="btn btn-primary"
              :disabled="participating"
              @click="handleParticipate"
            >
              <svg v-if="participating" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
              {{ participating ? t('lottery.participating') : t('lottery.participate') }}
            </button>
            <p v-else class="text-sm text-gray-400">{{ t('lottery.activityEnded') }}</p>
          </div>

          <!-- Already participated -->
          <div v-else class="space-y-3">
            <div class="rounded-lg p-4 text-center" :class="resultBgClass">
              <span v-if="myParticipation.is_winner === true" class="text-lg font-bold text-green-700 dark:text-green-400">
                {{ t('lottery.detail.won') }}
              </span>
              <span v-else-if="myParticipation.is_winner === false" class="text-base text-gray-600 dark:text-dark-400">
                {{ t('lottery.detail.lost') }}
              </span>
              <span v-else class="text-base text-yellow-700 dark:text-yellow-400">
                {{ t('lottery.detail.waitingDraw') }}
              </span>
            </div>
          </div>
        </div>

        <!-- Draw Results (if completed) -->
        <div v-if="activity.status === 'completed'" class="card p-6">
          <h2 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">{{ t('lottery.detail.results') }}</h2>
          <div class="grid grid-cols-2 gap-4 text-sm">
            <div class="rounded-lg bg-blue-50 p-4 text-center dark:bg-blue-900/20">
              <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">{{ activity.participant_count }}</p>
              <p class="mt-1 text-gray-500 dark:text-dark-400">{{ t('lottery.detail.totalParticipants') }}</p>
            </div>
            <div class="rounded-lg bg-green-50 p-4 text-center dark:bg-green-900/20">
              <p class="text-2xl font-bold text-green-600 dark:text-green-400">{{ activity.winner_count }}</p>
              <p class="mt-1 text-gray-500 dark:text-dark-400">{{ t('lottery.detail.totalWinners') }}</p>
            </div>
          </div>
        </div>
      </template>

      <!-- Not Found -->
      <div v-else class="card p-12 text-center">
        <p class="text-gray-500 dark:text-dark-400">{{ t('common.notFound') }}</p>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { getById, participate, getMyParticipations } from '@/api/lottery'
import type { LotteryActivity, LotteryParticipant } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import { formatDateTime } from '@/utils/format'
import LotteryShareQRCode from '@/components/lottery/LotteryShareQRCode.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const activity = ref<LotteryActivity | null>(null)
const myParticipation = ref<LotteryParticipant | null>(null)
const loading = ref(true)
const participating = ref(false)

const statusClass = computed(() => {
  const map: Record<string, string> = {
    pending: 'bg-white/20 text-white',
    active: 'bg-green-400/20 text-green-100',
    drawing: 'bg-yellow-400/20 text-yellow-100',
    completed: 'bg-blue-400/20 text-blue-100',
    cancelled: 'bg-red-400/20 text-red-100',
    expired: 'bg-gray-400/20 text-gray-100'
  }
  return map[activity.value?.status || ''] || 'bg-white/20 text-white'
})

// Compute validity_days from draw_at and activity_end_at
const validityDays = computed(() => {
  if (!activity.value?.draw_at || !activity.value?.activity_end_at) return 0
  const drawAt = new Date(activity.value.draw_at).getTime()
  const endAt = new Date(activity.value.activity_end_at).getTime()
  return Math.round((endAt - drawAt) / (24 * 60 * 60 * 1000))
})

const resultBgClass = computed(() => {
  if (!myParticipation.value) return ''
  if (myParticipation.value.is_winner === true) return 'bg-green-50 dark:bg-green-900/20'
  if (myParticipation.value.is_winner === false) return 'bg-gray-50 dark:bg-dark-700'
  return 'bg-yellow-50 dark:bg-yellow-900/20'
})

async function loadActivity() {
  const id = Number(route.params.id)
  if (!id) return
  try {
    activity.value = await getById(id)
  } catch (error) {
    console.error('Failed to load activity:', error)
  }
}

async function loadMyParticipation() {
  const activityId = Number(route.params.id)
  if (!activityId) return
  try {
    // Load all participations and find the one for this activity
    const res = await getMyParticipations(1, 100)
    myParticipation.value = res.items.find(p => p.activity_id === activityId) || null
  } catch {
    // User might not have participated
  }
}

async function handleParticipate() {
  const id = Number(route.params.id)
  if (!id) return
  participating.value = true
  try {
    const result = await participate(id)
    myParticipation.value = result
    appStore.showSuccess(t('lottery.participateSuccess'))
    // Refresh activity to update participant_count
    await loadActivity()
  } catch (error: any) {
    const message = error.response?.data?.detail || t('lottery.participateFailed')
    appStore.showError(message)
  } finally {
    participating.value = false
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([loadActivity(), loadMyParticipation()])
  } finally {
    loading.value = false
  }

  // 自动参与：从分享页注册/登录后跳转回来时自动触发参与
  if (route.query.auto_join === '1' && !myParticipation.value && activity.value?.status === 'active') {
    try {
      await handleParticipate()
    } finally {
      // 无论成功失败都清除 auto_join 参数，避免刷新后重复触发
      router.replace({ path: route.path, query: {} })
    }
  }
})
</script>
