<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.avatar.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.avatar.description') }}
      </p>
    </div>

    <div class="flex flex-col gap-5 px-6 py-6 sm:flex-row sm:items-center">
      <UserAvatar :user="user" :src="previewDataUrl" size="2xl" shape="square" />

      <div class="min-w-0 flex-1 space-y-4">
        <div class="space-y-1">
          <p class="text-sm font-medium text-gray-900 dark:text-white">
            {{ previewDataUrl || currentAvatarUrl ? t('profile.avatar.ready') : t('profile.avatar.empty') }}
          </p>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('profile.avatar.hint') }}
          </p>
          <p class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('profile.avatar.formatsHint') }}
          </p>
          <p class="text-xs text-gray-400 dark:text-gray-500">
            {{ t('profile.avatar.gifHint') }}
          </p>
        </div>

        <div v-if="uploadMeta" class="rounded-xl bg-gray-50 px-4 py-3 text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300">
          {{ uploadMeta }}
        </div>

        <div v-if="errorMessage" class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400">
          {{ errorMessage }}
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <input
            ref="fileInputRef"
            type="file"
            accept="image/png,image/jpeg,image/webp,image/gif"
            class="hidden"
            @change="handleFileChange"
          />

          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="isBusy"
            @click="fileInputRef?.click()"
          >
            <Icon v-if="isBusy" name="refresh" size="sm" class="mr-1.5 animate-spin" />
            <Icon v-else name="upload" size="sm" class="mr-1.5" />
            {{ isUploading ? t('profile.avatar.uploading') : t('profile.avatar.selectImage') }}
          </button>

          <button
            v-if="previewDataUrl || currentAvatarUrl"
            type="button"
            class="btn btn-secondary btn-sm text-red-600 hover:text-red-700 dark:text-red-400"
            :disabled="isBusy"
            @click="handleRemoveAvatar"
          >
            <Icon v-if="isRemoving" name="refresh" size="sm" class="mr-1.5 animate-spin" />
            <Icon v-else name="trash" size="sm" class="mr-1.5" />
            {{ t('profile.avatar.remove') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { userAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import UserAvatar from '@/components/common/UserAvatar.vue'
import Icon from '@/components/icons/Icon.vue'
import type { User } from '@/types'
import { resolveUserAvatarUrl } from './profileUser'

const MAX_AVATAR_BYTES = 100 * 1024
const QUALITY_STEPS = [0.92, 0.82, 0.72, 0.64, 0.56, 0.48, 0.4, 0.32]
const SCALE_STEPS = [1, 0.92, 0.84, 0.76, 0.68, 0.6, 0.52, 0.44]

const props = defineProps<{
  user: User | null
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const fileInputRef = ref<HTMLInputElement | null>(null)
const previewDataUrl = ref<string | null>(null)
const uploadMeta = ref<string>('')
const errorMessage = ref('')
const isUploading = ref(false)
const isRemoving = ref(false)

const currentAvatarUrl = computed(() => resolveUserAvatarUrl(props.user))
const isBusy = computed(() => isUploading.value || isRemoving.value)

watch(currentAvatarUrl, (value) => {
  if (value) {
    previewDataUrl.value = null
  }
})

function syncUser(updatedUser: User): void {
  authStore.user = updatedUser
  localStorage.setItem('auth_user', JSON.stringify(updatedUser))
}

function clearInputValue(): void {
  if (fileInputRef.value) {
    fileInputRef.value.value = ''
  }
}

function formatKB(bytes: number): string {
  return `${(bytes / 1024).toFixed(1)} KB`
}

function getErrorMessage(error: unknown, fallback: string): string {
  return (
    (error as any)?.response?.data?.detail ||
    (error as any)?.response?.data?.message ||
    (error as any)?.message ||
    fallback
  )
}

function loadImage(dataUrl: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => resolve(image)
    image.onerror = () => reject(new Error(t('profile.avatar.readFailed')))
    image.src = dataUrl
  })
}

function fileToDataUrl(file: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(new Error(t('profile.avatar.readFailed')))
    reader.readAsDataURL(file)
  })
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality?: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (!blob) {
        reject(new Error(t('profile.avatar.compressFailed')))
        return
      }
      resolve(blob)
    }, type, quality)
  })
}

async function compressStaticImage(file: File): Promise<File> {
  const sourceDataUrl = await fileToDataUrl(file)
  const image = await loadImage(sourceDataUrl)
  const canvas = document.createElement('canvas')
  const ctx = canvas.getContext('2d')

  if (!ctx) {
    throw new Error(t('profile.avatar.compressFailed'))
  }

  for (const scale of SCALE_STEPS) {
    const width = Math.max(1, Math.round(image.naturalWidth * scale))
    const height = Math.max(1, Math.round(image.naturalHeight * scale))
    canvas.width = width
    canvas.height = height
    ctx.clearRect(0, 0, width, height)
    ctx.drawImage(image, 0, 0, width, height)

    for (const quality of QUALITY_STEPS) {
      const blob = await canvasToBlob(canvas, 'image/webp', quality)
      if (blob.size <= MAX_AVATAR_BYTES) {
        return new File([blob], file.name.replace(/\.[^.]+$/, '') + '.webp', {
          type: 'image/webp'
        })
      }
    }
  }

  throw new Error(t('profile.avatar.compressTooLarge'))
}

async function prepareAvatarFile(file: File): Promise<File> {
  if (!file.type.startsWith('image/')) {
    throw new Error(t('profile.avatar.invalidType'))
  }

  if (file.type === 'image/gif') {
    if (file.size > MAX_AVATAR_BYTES) {
      throw new Error(t('profile.avatar.gifTooLarge'))
    }
    return file
  }

  if (file.size <= MAX_AVATAR_BYTES) {
    return file
  }

  return compressStaticImage(file)
}

async function handleFileChange(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]

  clearInputValue()
  errorMessage.value = ''
  uploadMeta.value = ''

  if (!file) {
    return
  }

  isUploading.value = true

  try {
    const preparedFile = await prepareAvatarFile(file)
    const preparedDataUrl = await fileToDataUrl(preparedFile)

    previewDataUrl.value = preparedDataUrl
    uploadMeta.value =
      preparedFile.size === file.size
        ? t('profile.avatar.sizeReady', { size: formatKB(preparedFile.size) })
        : t('profile.avatar.compressedReady', {
            from: formatKB(file.size),
            to: formatKB(preparedFile.size)
          })

    const updatedUser = await userAPI.uploadAvatar(preparedFile)
    if (!resolveUserAvatarUrl(updatedUser)) {
      updatedUser.avatar_url = preparedDataUrl
    }
    syncUser(updatedUser)
    appStore.showSuccess(t('profile.avatar.uploadSuccess'))
  } catch (error) {
    previewDataUrl.value = null
    uploadMeta.value = ''
    errorMessage.value = getErrorMessage(error, t('profile.avatar.uploadFailed'))
  } finally {
    isUploading.value = false
  }
}

async function handleRemoveAvatar(): Promise<void> {
  errorMessage.value = ''
  uploadMeta.value = ''
  isRemoving.value = true

  try {
    const updatedUser = await userAPI.removeAvatar()
    updatedUser.avatar_url = null
    updatedUser.avatar_thumbnail_url = null
    syncUser(updatedUser)
    previewDataUrl.value = null
    appStore.showSuccess(t('profile.avatar.removeSuccess'))
  } catch (error) {
    errorMessage.value = getErrorMessage(error, t('profile.avatar.removeFailed'))
  } finally {
    isRemoving.value = false
  }
}

onBeforeUnmount(() => {
  clearInputValue()
})
</script>

