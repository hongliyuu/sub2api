<template>
  <AppLayout>
    <div class="mx-auto max-w-3xl space-y-6">
      <!-- Page Title -->
      <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
        {{ t('lottery.title') }}
      </h1>

      <!-- Loading State -->
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <template v-else>
        <!-- Active Activities Section -->
        <section class="space-y-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('lottery.activeActivities') }}
          </h2>

          <!-- Active Activity Cards -->
          <div v-if="activities.length > 0" class="space-y-4">
            <div
              v-for="activity in activities"
              :key="activity.id"
              class="card cursor-pointer overflow-hidden transition-shadow hover:shadow-lg"
              @click="goToDetail(activity.id)"
            >
              <div class="p-6">
                <div class="flex items-start justify-between gap-4">
                  <div class="min-w-0 flex-1">
                    <div class="flex items-center gap-2">
                      <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                        {{ activity.title }}
                      </h3>
                      <span
                        :class="[
                          'badge flex-shrink-0',
                          activity.status === 'active'
                            ? 'badge-success'
                            : activity.status === 'completed'
                              ? 'badge-info'
                              : activity.status === 'drawing'
                                ? 'badge-warning'
                                : 'badge-secondary'
                        ]"
                      >
                        {{ getStatusLabel(activity.status) }}
                      </span>
                    </div>
                    <p
                      v-if="activity.description"
                      class="mt-1 line-clamp-2 text-sm text-gray-500 dark:text-dark-400"
                    >
                      {{ activity.description }}
                    </p>

                    <!-- Activity Meta -->
                    <div class="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
                      <span class="flex items-center gap-1">
                        <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        {{ t('lottery.drawTime') }}: {{ formatDateTime(activity.draw_at) }}
                      </span>
                      <span class="flex items-center gap-1">
                        <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
                        </svg>
                        {{ t('lottery.participantCount') }}: {{ activity.participant_count }}
                      </span>
                      <span class="flex items-center gap-1">
                        <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                          <path stroke-linecap="round" stroke-linejoin="round" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                        </svg>
                        {{ t('lottery.winRate') }}: {{ (activity.base_win_rate * 100).toFixed(0) }}%
                      </span>
                    </div>
                  </div>

                  <!-- Participate Button / Badge -->
                  <div class="flex-shrink-0">
                    <button
                      v-if="!participatedIds.has(activity.id) && activity.status === 'active'"
                      class="btn btn-primary text-sm"
                      :disabled="participatingId === activity.id"
                      @click.stop="handleParticipate(activity.id)"
                    >
                      <svg
                        v-if="participatingId === activity.id"
                        class="-ml-1 mr-1.5 h-4 w-4 animate-spin"
                        fill="none"
                        viewBox="0 0 24 24"
                      >
                        <circle
                          class="opacity-25"
                          cx="12"
                          cy="12"
                          r="10"
                          stroke="currentColor"
                          stroke-width="4"
                        ></circle>
                        <path
                          class="opacity-75"
                          fill="currentColor"
                          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                        ></path>
                      </svg>
                      {{
                        participatingId === activity.id
                          ? t('lottery.participating')
                          : t('lottery.participate')
                      }}
                    </button>
                    <span
                      v-else-if="participatedIds.has(activity.id)"
                      class="badge badge-success"
                    >
                      {{ t('lottery.participated') }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- No Active Activities Empty State -->
          <div v-else class="card p-12 text-center">
            <div
              class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <svg
                class="h-8 w-8 text-gray-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M21 11.25v8.25a1.5 1.5 0 01-1.5 1.5H5.25a1.5 1.5 0 01-1.5-1.5v-8.25M12 4.875A2.625 2.625 0 109.375 7.5H12m0-2.625V7.5m0-2.625A2.625 2.625 0 1114.625 7.5H12m0 0V21m-8.625-9.75h18c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125h-18c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"
                />
              </svg>
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('lottery.noActiveActivities') }}
            </p>
          </div>
        </section>

        <!-- My Participations Section -->
        <section class="space-y-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('lottery.myParticipations') }}
          </h2>

          <!-- Loading Participations -->
          <div v-if="loadingParticipations" class="flex justify-center py-8">
            <div
              class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
            ></div>
          </div>

          <!-- Participation List -->
          <div v-else-if="participations.length > 0" class="space-y-3">
            <div
              v-for="item in participations"
              :key="item.id"
              class="card cursor-pointer overflow-hidden transition-shadow hover:shadow-md"
              @click="goToDetail(item.activity_id)"
            >
              <div class="flex items-center justify-between p-4">
                <div class="min-w-0 flex-1">
                  <p class="truncate text-sm font-medium text-gray-900 dark:text-white">
                    {{ getActivityTitle(item.activity_id) }}
                  </p>
                  <div class="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
                    <span>{{ formatDateTime(item.participated_at) }}</span>
                    <span class="rounded bg-gray-100 px-1.5 py-0.5 dark:bg-dark-700">
                      {{ item.user_category }}
                    </span>
                  </div>
                </div>
                <div class="ml-4 flex-shrink-0">
                  <span
                    v-if="item.is_winner === true"
                    class="badge badge-success"
                  >
                    {{ t('lottery.detail.won') }}
                  </span>
                  <span
                    v-else-if="item.is_winner === false"
                    class="badge badge-danger"
                  >
                    {{ t('lottery.detail.lost') }}
                  </span>
                  <span
                    v-else
                    class="badge badge-warning"
                  >
                    {{ t('lottery.detail.waitingDraw') }}
                  </span>
                </div>
              </div>
            </div>

            <!-- Pagination -->
            <div
              v-if="participationPages > 1"
              class="flex items-center justify-center gap-2 pt-4"
            >
              <button
                class="btn btn-secondary text-sm"
                :disabled="participationPage <= 1"
                @click="loadParticipations(participationPage - 1)"
              >
                &laquo;
              </button>
              <span class="text-sm text-gray-500 dark:text-dark-400">
                {{ participationPage }} / {{ participationPages }}
              </span>
              <button
                class="btn btn-secondary text-sm"
                :disabled="participationPage >= participationPages"
                @click="loadParticipations(participationPage + 1)"
              >
                &raquo;
              </button>
            </div>
          </div>

          <!-- No Participations Empty State -->
          <div v-else class="card p-12 text-center">
            <div
              class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <svg
                class="h-8 w-8 text-gray-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25z"
                />
              </svg>
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('lottery.noParticipations') }}
            </p>
          </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { getActive, getMyParticipations, participate } from '@/api/lottery'
import type { LotteryActivity, LotteryParticipant } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

// Active activities
const activities = ref<LotteryActivity[]>([])
const loading = ref(true)

// My participations
const participations = ref<LotteryParticipant[]>([])
const loadingParticipations = ref(false)
const participationPage = ref(1)
const participationPages = ref(1)

// Participate state
const participatingId = ref<number | null>(null)

// Track which activities the user has already participated in
const participatedIds = computed(() => {
  const ids = new Set<number>()
  for (const p of participations.value) {
    ids.add(p.activity_id)
  }
  return ids
})

// Map activity_id -> title for display in participation list
const activityTitleMap = computed(() => {
  const map = new Map<number, string>()
  for (const a of activities.value) {
    map.set(a.id, a.title)
  }
  return map
})

function getActivityTitle(activityId: number): string {
  return activityTitleMap.value.get(activityId) || `#${activityId}`
}

function getStatusLabel(status: LotteryActivity['status']): string {
  const map: Record<string, string> = {
    pending: t('lottery.status.pending', 'Pending'),
    active: t('lottery.status.active', 'Active'),
    drawing: t('lottery.status.drawing', 'Drawing'),
    completed: t('lottery.status.completed', 'Completed'),
    cancelled: t('lottery.status.cancelled', 'Cancelled'),
    expired: t('lottery.status.expired', 'Expired')
  }
  return map[status] || status
}

function goToDetail(activityId: number) {
  router.push(`/lottery/${activityId}`)
}

async function handleParticipate(activityId: number) {
  participatingId.value = activityId
  try {
    const result = await participate(activityId)
    // Add to participations list
    participations.value.unshift(result)
    appStore.showSuccess(t('lottery.participateSuccess'))
    // Refresh activity list to update participant_count
    await loadActivities()
  } catch (error: any) {
    const message = error.response?.data?.detail || t('lottery.participateFailed')
    appStore.showError(message)
  } finally {
    participatingId.value = null
  }
}

async function loadActivities() {
  try {
    activities.value = await getActive()
  } catch (error) {
    console.error('Failed to load active activities:', error)
  }
}

async function loadParticipations(page: number = 1) {
  loadingParticipations.value = true
  try {
    const res = await getMyParticipations(page, 10)
    participations.value = res.items
    participationPage.value = res.page
    participationPages.value = res.pages
  } catch (error) {
    console.error('Failed to load participations:', error)
  } finally {
    loadingParticipations.value = false
  }
}

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([loadActivities(), loadParticipations()])
  } finally {
    loading.value = false
  }
})
</script>
