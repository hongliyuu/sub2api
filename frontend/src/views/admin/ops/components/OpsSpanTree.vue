<script setup lang="ts">
import { ref } from 'vue'
import type { OpsSpan } from '@/api/admin/ops'

defineProps<{
  spans: OpsSpan[]
}>()

const expanded = ref(true)

function statusClass(status?: string): string {
  switch (status) {
    case 'error':   return 'text-red-500'
    case 'skipped': return 'text-gray-400'
    default:        return 'text-emerald-500'
  }
}

function statusIcon(status?: string): string {
  switch (status) {
    case 'error':   return '✕'
    case 'skipped': return '—'
    default:        return '✓'
  }
}
</script>

<template>
  <div class="font-mono text-xs">
    <button
      class="mb-1 flex items-center gap-1 text-[11px] font-bold uppercase tracking-wider text-gray-400 hover:text-gray-600"
      @click="expanded = !expanded"
    >
      <span>{{ expanded ? '▾' : '▸' }}</span>
      <span>Span 追踪 ({{ spans.length }})</span>
    </button>

    <div v-if="expanded" class="space-y-0.5 pl-3">
      <div
        v-for="(span, i) in spans"
        :key="i"
        class="flex items-start gap-2 py-0.5"
      >
        <span :class="statusClass(span.status)" class="w-3 flex-shrink-0 text-center">
          {{ statusIcon(span.status) }}
        </span>
        <span class="w-32 flex-shrink-0 text-gray-700 dark:text-gray-300">
          {{ span.name }}
        </span>
        <span class="w-16 flex-shrink-0 text-right text-gray-500">
          {{ span.duration_ms }}ms
        </span>
        <span
          v-if="span.attrs && Object.keys(span.attrs).length > 0"
          class="flex flex-wrap gap-x-2 text-[10px] text-gray-400"
        >
          <template v-for="(val, key) in span.attrs" :key="key">
            <span>{{ key }}=<span class="text-gray-600 dark:text-gray-300">{{ val }}</span></span>
          </template>
        </span>
      </div>
    </div>
  </div>
</template>
