<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <Select v-model="filters.status" :options="filterStatusOptions" class="w-36" @change="loadActivities" />
          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button @click="loadActivities" :disabled="loading" class="btn btn-secondary" :title="t('common.refresh')">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button @click="showCreateDialog = true" class="btn btn-primary">
              <Icon name="plus" size="md" class="mr-1" />
              {{ t('admin.lottery.createActivity') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="activities" :loading="loading">
          <template #cell-status="{ value }">
            <span :class="['badge', statusBadgeClass[value as LotteryActivity['status']]]">
              {{ t(`admin.lottery.status.${value}`) }}
            </span>
          </template>
          <template #cell-draw_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ value ? formatDateTime(value) : '-' }}</span>
          </template>
          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center space-x-1">
              <button @click="copyShareLink(row)" class="flex items-center rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400" :title="t('admin.lottery.copyShareLink')">
                <Icon name="link" size="sm" />
              </button>
              <button @click="handleViewDetail(row)" class="flex items-center rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400" :title="t('admin.lottery.viewDetail')">
                <Icon name="eye" size="sm" />
              </button>
              <button v-if="row.status === 'active'" @click="handleDraw(row)" class="flex items-center rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-yellow-50 hover:text-yellow-600 dark:hover:bg-yellow-900/20 dark:hover:text-yellow-400" :title="t('admin.lottery.draw')">
                <Icon name="zap" size="sm" />
              </button>
              <button v-if="row.status === 'pending' || row.status === 'active'" @click="handleCancel(row)" class="flex items-center rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400" :title="t('common.cancel')">
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination v-if="pagination.total > 0" :page="pagination.page" :total="pagination.total" :page-size="pagination.page_size" @update:page="handlePageChange" @update:pageSize="handlePageSizeChange" />
      </template>
    </TablePageLayout>

    <!-- Create Activity Dialog -->
    <BaseDialog :show="showCreateDialog" :title="t('admin.lottery.createActivity')" width="normal" @close="showCreateDialog = false">
      <form id="create-lottery-form" @submit.prevent="handleCreate" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.lottery.fields.title') }} *</label>
          <input v-model="createForm.title" type="text" required class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.lottery.fields.description') }}</label>
          <textarea v-model="createForm.description" rows="3" class="input"></textarea>
        </div>
        <div>
          <label class="input-label">{{ t('admin.lottery.fields.drawAt') }}</label>
          <input v-model="createForm.draw_at_str" type="datetime-local" class="input" />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.lottery.fields.minParticipants') }}</label>
            <input v-model.number="createForm.min_participants" type="number" min="1" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.lottery.fields.baseWinRate') }}</label>
            <input v-model.number="createForm.base_win_rate" type="number" min="0" max="1" step="0.01" class="input" />
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.lottery.fields.winnerDiscount') }}</label>
            <input v-model.number="createForm.winner_discount_percent" type="number" min="1" max="100" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.lottery.fields.loserCoupon') }}</label>
            <input v-model.number="createForm.loser_coupon_amount" type="number" min="0" class="input" />
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" @click="showCreateDialog = false" class="btn btn-secondary">{{ t('common.cancel') }}</button>
          <button type="submit" form="create-lottery-form" :disabled="creating" class="btn btn-primary">
            {{ creating ? t('common.creating') : t('common.create') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Detail Dialog (Participants) -->
    <BaseDialog :show="showDetailDialog" :title="detailActivity?.title ?? ''" width="wide" @close="showDetailDialog = false">
      <!-- Share QR Code -->
      <div v-if="detailActivity?.share_code" class="mb-6 flex flex-col items-center rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700">
        <p class="mb-2 text-sm font-medium text-gray-700 dark:text-dark-300">{{ t('lottery.share.qrCode') }}</p>
        <LotteryShareQRCode :share-code="detailActivity.share_code" :size="160" />
      </div>
      <div v-if="participantsLoading" class="flex items-center justify-center py-8">
        <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
      </div>
      <div v-else-if="participants.length === 0" class="py-8 text-center text-gray-500 dark:text-gray-400">
        {{ t('admin.lottery.noParticipants') }}
      </div>
      <div v-else class="space-y-3">
        <div v-for="p in participants" :key="p.id" class="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <div class="flex items-center gap-3">
            <div class="flex h-8 w-8 items-center justify-center rounded-full bg-blue-100 dark:bg-blue-900/30">
              <Icon name="user" size="sm" class="text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ p.user?.email || t('admin.lottery.userPrefix', { id: p.user_id }) }}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.lottery.category') }}: {{ p.user_category }} | {{ formatDateTime(p.participated_at) }}
              </p>
            </div>
          </div>
          <div class="text-right">
            <span v-if="p.is_winner === true" class="badge badge-success">{{ t('admin.lottery.winner') }}</span>
            <span v-else-if="p.is_winner === false" class="badge badge-gray">{{ t('admin.lottery.loser') }}</span>
            <span v-else class="badge badge-warning">{{ t('admin.lottery.pending') }}</span>
          </div>
        </div>
        <Pagination v-if="participantsTotal > participantsPageSize" :page="participantsPage" :total="participantsTotal" :page-size="participantsPageSize" :page-size-options="[10, 20, 50]" @update:page="handleParticipantsPageChange" @update:page-size="(size: number) => { participantsPageSize = size; participantsPage = 1; loadParticipants() }" class="mt-4" />
      </div>
      <template #footer>
        <div class="flex justify-end">
          <button type="button" @click="showDetailDialog = false" class="btn btn-secondary">{{ t('common.close') }}</button>
        </div>
      </template>
    </BaseDialog>

    <!-- Draw Confirmation -->
    <ConfirmDialog :show="showDrawDialog" :title="t('admin.lottery.drawConfirmTitle')" :message="t('admin.lottery.drawConfirmMessage')" :confirm-text="t('admin.lottery.draw')" :cancel-text="t('common.cancel')" @confirm="confirmDraw" @cancel="showDrawDialog = false" />

    <!-- Cancel Confirmation -->
    <ConfirmDialog :show="showCancelDialog" :title="t('admin.lottery.cancelConfirmTitle')" :message="t('admin.lottery.cancelConfirmMessage')" :confirm-text="t('common.confirm')" :cancel-text="t('common.cancel')" danger @confirm="confirmCancel" @cancel="showCancelDialog = false" />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { adminAPI } from '@/api/admin'
import { formatDateTime } from '@/utils/format'
import type { LotteryActivity, LotteryParticipant, CreateLotteryActivityRequest } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import LotteryShareQRCode from '@/components/lottery/LotteryShareQRCode.vue'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const activities = ref<LotteryActivity[]>([])
const loading = ref(false)
const creating = ref(false)
const filters = reactive({ status: '' })
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

// Dialogs
const showCreateDialog = ref(false)
const showDetailDialog = ref(false)
const showDrawDialog = ref(false)
const showCancelDialog = ref(false)
const detailActivity = ref<LotteryActivity | null>(null)
const drawingActivity = ref<LotteryActivity | null>(null)
const cancellingActivity = ref<LotteryActivity | null>(null)

// Participants
const participants = ref<LotteryParticipant[]>([])
const participantsLoading = ref(false)
const participantsPage = ref(1)
const participantsPageSize = ref(20)
const participantsTotal = ref(0)

// Default draw_at: tomorrow 10:00
const getDefaultDrawAt = () => {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  d.setHours(10, 0, 0, 0)
  return d.toISOString().slice(0, 16)
}

const createForm = reactive({
  title: '',
  description: '',
  draw_at_str: getDefaultDrawAt(),
  min_participants: 10,
  base_win_rate: 0.1,
  winner_discount_percent: 95,
  loser_coupon_amount: 30
})

const statusBadgeClass: Record<string, string> = {
  pending: 'badge-gray',
  active: 'badge-success',
  drawing: 'badge-warning',
  completed: 'badge-primary',
  cancelled: 'badge-danger',
  expired: 'badge-gray'
}

const filterStatusOptions = computed(() => [
  { value: '', label: t('admin.lottery.allStatus') },
  { value: 'pending', label: t('admin.lottery.status.pending') },
  { value: 'active', label: t('admin.lottery.status.active') },
  { value: 'drawing', label: t('admin.lottery.status.drawing') },
  { value: 'completed', label: t('admin.lottery.status.completed') },
  { value: 'cancelled', label: t('admin.lottery.status.cancelled') },
  { value: 'expired', label: t('admin.lottery.status.expired', 'Expired') }
])

const columns = computed<Column[]>(() => [
  { key: 'title', label: t('admin.lottery.columns.title') },
  { key: 'status', label: t('admin.lottery.columns.status') },
  { key: 'participant_count', label: t('admin.lottery.columns.participantCount'), sortable: true },
  { key: 'winner_count', label: t('admin.lottery.columns.winnerCount'), sortable: true },
  { key: 'draw_at', label: t('admin.lottery.columns.drawAt'), sortable: true },
  { key: 'created_at', label: t('admin.lottery.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.lottery.columns.actions') }
])

// Load activities
let abortController: AbortController | null = null

const loadActivities = async () => {
  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  loading.value = true
  try {
    const res = await adminAPI.lottery.list(pagination.page, pagination.page_size, { status: filters.status || undefined })
    if (ctrl.signal.aborted) return
    activities.value = res.items
    pagination.total = res.total
  } catch (error: any) {
    if (ctrl.signal.aborted || error?.name === 'AbortError') return
    appStore.showError(t('admin.lottery.failedToLoad'))
  } finally {
    if (abortController === ctrl && !ctrl.signal.aborted) {
      loading.value = false
      abortController = null
    }
  }
}

const handlePageChange = (page: number) => { pagination.page = page; loadActivities() }
const handlePageSizeChange = (size: number) => { pagination.page_size = size; pagination.page = 1; loadActivities() }

// Create
const handleCreate = async () => {
  creating.value = true
  try {
    const req: CreateLotteryActivityRequest = {
      title: createForm.title,
      description: createForm.description || undefined,
      draw_at: createForm.draw_at_str ? Math.floor(new Date(createForm.draw_at_str).getTime() / 1000) : undefined,
      min_participants: createForm.min_participants,
      base_win_rate: createForm.base_win_rate,
      winner_discount_percent: createForm.winner_discount_percent,
      loser_coupon_amount: createForm.loser_coupon_amount
    }
    await adminAPI.lottery.create(req)
    appStore.showSuccess(t('admin.lottery.activityCreated'))
    showCreateDialog.value = false
    resetCreateForm()
    loadActivities()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.lottery.failedToCreate'))
  } finally {
    creating.value = false
  }
}

const resetCreateForm = () => {
  Object.assign(createForm, {
    title: '', description: '', draw_at_str: getDefaultDrawAt(),
    min_participants: 10, base_win_rate: 0.1, winner_discount_percent: 95, loser_coupon_amount: 30
  })
}

const copyShareLink = (activity: LotteryActivity) => {
  copyToClipboard(`${window.location.origin}/lottery/share/${activity.share_code}`, t('admin.lottery.shareLinkCopied'))
}

// Detail / Participants
const handleViewDetail = async (activity: LotteryActivity) => {
  detailActivity.value = activity
  showDetailDialog.value = true
  participantsPage.value = 1
  await loadParticipants()
}

const loadParticipants = async () => {
  if (!detailActivity.value) return
  participantsLoading.value = true
  participants.value = []
  try {
    const res = await adminAPI.lottery.listParticipants(detailActivity.value.id, participantsPage.value, participantsPageSize.value)
    participants.value = res.items
    participantsTotal.value = res.total
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.lottery.failedToLoadParticipants'))
  } finally {
    participantsLoading.value = false
  }
}

const handleParticipantsPageChange = (page: number) => { participantsPage.value = page; loadParticipants() }

// Draw
const handleDraw = (activity: LotteryActivity) => { drawingActivity.value = activity; showDrawDialog.value = true }

const confirmDraw = async () => {
  if (!drawingActivity.value) return
  try {
    const result = await adminAPI.lottery.draw(drawingActivity.value.id)
    if (result.below_min_participant) {
      appStore.showError(t('admin.lottery.belowMinParticipants'))
    } else {
      appStore.showSuccess(t('admin.lottery.drawSuccess', { winners: result.winner_count, total: result.total_participants }))
    }
    showDrawDialog.value = false
    drawingActivity.value = null
    loadActivities()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.lottery.failedToDraw'))
  }
}

// Cancel
const handleCancel = (activity: LotteryActivity) => { cancellingActivity.value = activity; showCancelDialog.value = true }

const confirmCancel = async () => {
  if (!cancellingActivity.value) return
  try {
    await adminAPI.lottery.cancel(cancellingActivity.value.id)
    appStore.showSuccess(t('admin.lottery.activityCancelled'))
    showCancelDialog.value = false
    cancellingActivity.value = null
    loadActivities()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.lottery.failedToCancel'))
  }
}

onMounted(() => { loadActivities() })
onUnmounted(() => { abortController?.abort() })
</script>
