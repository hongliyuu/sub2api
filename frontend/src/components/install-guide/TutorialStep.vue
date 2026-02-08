<template>
  <div>
    <h3 class="mb-2 flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white sm:mb-3 sm:text-base">
      <span
        class="flex h-5 w-5 items-center justify-center rounded-full text-xs font-semibold sm:h-6 sm:w-6"
        :class="badgeClasses"
      >
        {{ step }}
      </span>
      {{ title }}
    </h3>
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  step: number
  title: string
  color?: string
}>(), {
  color: 'primary'
})

const badgeClassesMap: Record<string, string> = {
  primary: 'bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400',
  emerald: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-400',
  blue: 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
}

const badgeClasses = computed(() => badgeClassesMap[props.color] || badgeClassesMap.primary)
</script>
