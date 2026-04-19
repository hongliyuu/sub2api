<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4 py-6"
    role="dialog"
    aria-modal="true"
    aria-labelledby="identity-adoption-title"
  >
    <div class="w-full max-w-lg rounded-2xl bg-white p-6 shadow-2xl dark:bg-dark-800">
      <div class="space-y-2">
        <p class="text-sm font-medium uppercase tracking-wide text-gray-500 dark:text-dark-300">
          {{ providerLabel }}
        </p>
        <h2 id="identity-adoption-title" class="text-xl font-semibold text-gray-900 dark:text-white">
          Review your imported profile
        </h2>
        <p class="text-sm text-gray-600 dark:text-dark-300">
          Choose whether to adopt the provider profile details for this first binding.
        </p>
      </div>

      <div class="mt-6 space-y-4">
        <div
          v-if="displayName || avatarUrl"
          class="flex items-center gap-4 rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700"
        >
          <div
            class="flex h-14 w-14 items-center justify-center overflow-hidden rounded-full bg-gray-200 text-sm font-semibold text-gray-600 dark:bg-dark-600 dark:text-dark-200"
          >
            <img
              v-if="avatarUrl"
              :src="avatarUrl"
              alt="Provider avatar preview"
              class="h-full w-full object-cover"
            />
            <span v-else>{{ avatarFallback }}</span>
          </div>

          <div class="min-w-0 flex-1">
            <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-dark-300">
              Provider suggestion
            </p>
            <p class="truncate text-base font-medium text-gray-900 dark:text-white">
              {{ displayName || 'No nickname provided' }}
            </p>
            <p class="truncate text-sm text-gray-500 dark:text-dark-300">
              {{ avatarUrl || 'No avatar URL provided' }}
            </p>
          </div>
        </div>

        <label
          class="flex items-start gap-3 rounded-xl border border-gray-200 p-4 text-sm text-gray-700 dark:border-dark-600 dark:text-dark-200"
        >
          <input
            v-model="adoptDisplayName"
            type="checkbox"
            class="mt-1 h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
            :disabled="!displayName"
          />
          <span>
            <span class="block font-medium text-gray-900 dark:text-white">Use provider nickname</span>
            <span class="block text-gray-500 dark:text-dark-300">
              {{ displayName || 'This provider did not return a nickname.' }}
            </span>
          </span>
        </label>

        <label
          class="flex items-start gap-3 rounded-xl border border-gray-200 p-4 text-sm text-gray-700 dark:border-dark-600 dark:text-dark-200"
        >
          <input
            v-model="adoptAvatar"
            type="checkbox"
            class="mt-1 h-4 w-4 rounded border-gray-300 text-brand-600 focus:ring-brand-500"
            :disabled="!avatarUrl"
          />
          <span>
            <span class="block font-medium text-gray-900 dark:text-white">Use provider avatar</span>
            <span class="block text-gray-500 dark:text-dark-300">
              {{ avatarUrl || 'This provider did not return an avatar URL.' }}
            </span>
          </span>
        </label>
      </div>

      <div class="mt-6 flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
        <button type="button" class="btn btn-secondary" @click="handleSkip">Keep current profile</button>
        <button type="button" class="btn btn-primary" @click="handleConfirm">Save choices</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  provider: string
  displayName?: string | null
  avatarUrl?: string | null
}>()

const emit = defineEmits<{
  confirm: [decision: { adoptDisplayName: boolean; adoptAvatar: boolean }]
  skip: []
}>()

const adoptDisplayName = ref(Boolean(props.displayName))
const adoptAvatar = ref(Boolean(props.avatarUrl))

watch(
  () => props.displayName,
  (value) => {
    adoptDisplayName.value = Boolean(value)
  },
  { immediate: true }
)

watch(
  () => props.avatarUrl,
  (value) => {
    adoptAvatar.value = Boolean(value)
  },
  { immediate: true }
)

const providerLabel = computed(() => {
  if (!props.provider) return 'Third-party identity'
  return props.provider
    .split('_')
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(' ')
})

const avatarFallback = computed(() => {
  const source = props.displayName?.trim() || props.provider.trim() || '?'
  return source.charAt(0).toUpperCase()
})

function handleConfirm() {
  emit('confirm', {
    adoptDisplayName: Boolean(props.displayName) && adoptDisplayName.value,
    adoptAvatar: Boolean(props.avatarUrl) && adoptAvatar.value
  })
}

function handleSkip() {
  emit('skip')
}
</script>
