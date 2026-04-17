<template>
  <div class="card overflow-hidden">
    <div
      class="border-b border-gray-100 bg-gradient-to-r from-primary-500/10 to-primary-600/5 px-6 py-5 dark:border-dark-700 dark:from-primary-500/20 dark:to-primary-600/10"
    >
      <div class="flex items-center gap-4">
        <UserAvatar :user="user" size="lg" shape="square" class="shadow-lg shadow-primary-500/20" />
        <div class="min-w-0 flex-1">
          <h2 class="truncate text-lg font-semibold text-gray-900 dark:text-white">
            {{ displayName }}
          </h2>
          <p class="mt-1 truncate text-sm text-gray-500 dark:text-gray-400">
            {{ user?.email }}
          </p>
          <div class="mt-1 flex items-center gap-2">
            <span :class="['badge', user?.role === 'admin' ? 'badge-primary' : 'badge-gray']">
              {{ user?.role === 'admin' ? t('profile.administrator') : t('profile.user') }}
            </span>
            <span
              :class="['badge', user?.status === 'active' ? 'badge-success' : 'badge-danger']"
            >
              {{ user?.status }}
            </span>
          </div>
        </div>
      </div>
    </div>
    <div class="px-6 py-4">
      <div class="space-y-3">
        <div class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400">
          <Icon name="mail" size="sm" class="text-gray-400 dark:text-gray-500" />
          <span class="truncate">{{ user?.email }}</span>
        </div>
        <div
          v-if="user?.username"
          class="flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400"
        >
          <Icon name="user" size="sm" class="text-gray-400 dark:text-gray-500" />
          <span class="truncate">{{ user.username }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import UserAvatar from '@/components/common/UserAvatar.vue'
import Icon from '@/components/icons/Icon.vue'
import type { User } from '@/types'
import { getUserDisplayName } from './profileUser'

const props = defineProps<{
  user: User | null
}>()

const { t } = useI18n()
const displayName = computed(() => getUserDisplayName(props.user) || props.user?.email || 'User')
</script>
