<template>
  <div>
    <div class="mb-1 flex items-center gap-2">
      <Icon name="document" size="sm" class="text-gray-400" />
      <span class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ filename }}</span>
    </div>
    <CodeBlock :code="code" :language="language" />
  </div>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
import CodeBlock from '@/components/common/CodeBlock.vue'

withDefaults(defineProps<{
  filename: string
  code: string
  language?: string
}>(), {
  language: 'yaml'
})
</script>
