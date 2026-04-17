<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.bindings.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('profile.bindings.description') }}
      </p>
    </div>

    <div class="divide-y divide-gray-100 dark:divide-dark-700">
      <div
        v-for="binding in bindings"
        :key="binding.provider"
        class="flex flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="flex min-w-0 items-start gap-4">
          <div :class="providerIconClass(binding.provider)" class="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl text-sm font-semibold">
            <Icon
              v-if="binding.provider === 'email'"
              name="mail"
              size="sm"
              class="text-current"
            />
            <span v-else>{{ providerInitial(binding.provider) }}</span>
          </div>

          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h3 class="font-medium text-gray-900 dark:text-white">
                {{ t(`profile.bindings.providers.${binding.provider}`) }}
              </h3>
              <span :class="statusClass(binding)" class="badge">
                {{ statusLabel(binding) }}
              </span>
            </div>
            <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
              {{ bindingDescription(binding) }}
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
              {{ bindingHint(binding) }}
            </p>
          </div>
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <button
            v-if="canConnect(binding)"
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="Boolean(pendingProvider)"
            @click="startBinding(binding)"
          >
            <Icon name="link" size="sm" class="mr-1.5" />
            {{ t('profile.bindings.connect') }}
          </button>

          <button
            v-else-if="canDisconnect(binding)"
            type="button"
            class="btn btn-secondary btn-sm text-red-600 hover:text-red-700 dark:text-red-400"
            :disabled="Boolean(pendingProvider)"
            @click="disconnectBinding(binding)"
          >
            <Icon
              :name="pendingProvider === binding.provider ? 'refresh' : 'x'"
              size="sm"
              :class="['mr-1.5', pendingProvider === binding.provider && 'animate-spin']"
            />
            {{ t('profile.bindings.disconnect') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { userAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import Icon from '@/components/icons/Icon.vue'
import type { User, UserAccountBindingProvider } from '@/types'
import {
  formatBindingValue,
  resolveUserBindings,
  type ResolvedUserBinding
} from './profileUser'

const props = defineProps<{
  user: User | null
}>()

const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const pendingProvider = ref<UserAccountBindingProvider | null>(null)

const publicSettings = computed(() => appStore.cachedPublicSettings)
const bindingContractAvailable = computed(() => props.user?.account_bindings != null)
const bindings = computed(() => resolveUserBindings(props.user))

function syncUser(updatedUser: User): void {
  authStore.user = updatedUser
  localStorage.setItem('auth_user', JSON.stringify(updatedUser))
}

function providerInitial(provider: UserAccountBindingProvider): string {
  if (provider === 'linuxdo') return 'L'
  if (provider === 'wechat') return 'W'
  return 'E'
}

function providerIconClass(provider: UserAccountBindingProvider): string {
  if (provider === 'linuxdo') {
    return 'bg-orange-100 text-orange-600 dark:bg-orange-900/20 dark:text-orange-300'
  }
  if (provider === 'wechat') {
    return 'bg-green-100 text-green-600 dark:bg-green-900/20 dark:text-green-300'
  }
  return 'bg-primary-100 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300'
}

function statusClass(binding: ResolvedUserBinding): string {
  if (binding.bound) {
    return 'badge-success'
  }
  if (providerEnabled(binding.provider)) {
    return 'badge-gray'
  }
  return 'badge-warning'
}

function statusLabel(binding: ResolvedUserBinding): string {
  if (binding.bound) {
    return t('profile.bindings.bound')
  }
  if (providerEnabled(binding.provider)) {
    return t('profile.bindings.unbound')
  }
  return t('profile.bindings.unavailable')
}

function providerEnabled(provider: UserAccountBindingProvider): boolean {
  if (provider === 'email') {
    return true
  }
  if (provider === 'linuxdo') {
    return Boolean(publicSettings.value?.linuxdo_oauth_enabled)
  }
  return Boolean(publicSettings.value?.wechat_oauth_enabled)
}

function bindingDescription(binding: ResolvedUserBinding): string {
  const value = formatBindingValue(binding.provider, binding.value)

  if (binding.provider === 'email') {
    if (value && binding.verified === true) {
      return t('profile.bindings.emailVerifiedValue', { value })
    }
    if (value && binding.verified === false) {
      return t('profile.bindings.emailUnverifiedValue', { value })
    }
    return value || t('profile.bindings.emailPrimary')
  }

  if (binding.bound) {
    return value
      ? t('profile.bindings.connectedAs', { value })
      : t('profile.bindings.connected')
  }

  return t(`profile.bindings.providerDescriptions.${binding.provider}`)
}

function bindingHint(binding: ResolvedUserBinding): string {
  if (!providerEnabled(binding.provider) && binding.provider !== 'email') {
    return t('profile.bindings.providerDisabled')
  }

  if (!bindingContractAvailable.value && binding.provider !== 'email') {
    return t('profile.bindings.providerPending')
  }

  return t(`profile.bindings.providerHints.${binding.provider}`)
}

function canConnect(binding: ResolvedUserBinding): boolean {
  return (
    binding.provider !== 'email' &&
    providerEnabled(binding.provider) &&
    !binding.bound &&
    bindingContractAvailable.value &&
    !pendingProvider.value
  )
}

function canDisconnect(binding: ResolvedUserBinding): boolean {
  return (
    binding.provider !== 'email' &&
    binding.bound &&
    bindingContractAvailable.value &&
    binding.canDisconnect === true &&
    !pendingProvider.value
  )
}

function bindingConnectUrl(binding: ResolvedUserBinding): string {
  if (binding.connectUrl && binding.connectUrl.includes('intent=bind')) {
    return binding.connectUrl
  }

  const provider = binding.provider as Extract<UserAccountBindingProvider, 'linuxdo' | 'wechat'>
  const redirectTo = (() => {
    const current = route.fullPath || '/profile'
    if (current.includes('oauth_intent=bind')) {
      return current
    }
    return current.includes('?') ? `${current}&oauth_intent=bind` : `${current}?oauth_intent=bind`
  })()
  return userAPI.getOAuthBindingStartUrl(provider, redirectTo, 'bind')
}

function getErrorMessage(error: unknown, fallback: string): string {
  return (
    (error as any)?.response?.data?.detail ||
    (error as any)?.response?.data?.message ||
    (error as any)?.message ||
    fallback
  )
}

function startBinding(binding: ResolvedUserBinding): void {
  window.location.href = bindingConnectUrl(binding)
}

async function disconnectBinding(binding: ResolvedUserBinding): Promise<void> {
  pendingProvider.value = binding.provider

  try {
    const updatedUser = await userAPI.unbindAccount(binding.provider)
    syncUser(updatedUser)
    appStore.showSuccess(
      t('profile.bindings.disconnectSuccess', {
        provider: t(`profile.bindings.providers.${binding.provider}`)
      })
    )
  } catch (error) {
    appStore.showError(getErrorMessage(error, t('profile.bindings.disconnectFailed')))
  } finally {
    pendingProvider.value = null
  }
}
</script>
