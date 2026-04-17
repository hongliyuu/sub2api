<template>
  <div
    :class="[
      sizeClass,
      shapeClass,
      'flex shrink-0 items-center justify-center overflow-hidden bg-gradient-to-br from-primary-500 to-primary-600 text-white shadow-sm'
    ]"
  >
    <img
      v-if="imageSource && !imageFailed"
      :src="imageSource"
      :alt="altText"
      class="h-full w-full object-cover"
      @error="imageFailed = true"
    />
    <span v-else :class="textClass" class="font-semibold tracking-wide">
      {{ initials }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { User } from '@/types'
import {
  getUserDisplayName,
  getUserInitials,
  resolveUserAvatarUrl
} from '@/components/user/profile/profileUser'

const props = withDefaults(defineProps<{
  user?: User | null
  src?: string | null
  alt?: string
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl'
  shape?: 'circle' | 'square'
}>(), {
  user: null,
  src: null,
  alt: '',
  size: 'md',
  shape: 'square'
})

const imageFailed = ref(false)

const imageSource = computed(() => props.src?.trim() || resolveUserAvatarUrl(props.user))
const initials = computed(() => getUserInitials(props.user))
const altText = computed(() => props.alt || getUserDisplayName(props.user) || 'User avatar')

const sizeClass = computed(() => ({
  xs: 'h-6 w-6',
  sm: 'h-8 w-8',
  md: 'h-10 w-10',
  lg: 'h-16 w-16',
  xl: 'h-20 w-20',
  '2xl': 'h-24 w-24'
}[props.size]))

const textClass = computed(() => ({
  xs: 'text-[10px]',
  sm: 'text-xs',
  md: 'text-sm',
  lg: 'text-2xl',
  xl: 'text-3xl',
  '2xl': 'text-4xl'
}[props.size]))

const shapeClass = computed(() => props.shape === 'circle' ? 'rounded-full' : 'rounded-2xl')

watch(imageSource, () => {
  imageFailed.value = false
})
</script>

