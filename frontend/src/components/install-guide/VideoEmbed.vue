<template>
  <div
    v-if="embedUrl"
    class="relative w-full overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700"
    style="padding-bottom: 56.25%"
  >
    <iframe
      :src="embedUrl"
      :title="title"
      class="absolute inset-0 h-full w-full"
      frameborder="0"
      allowfullscreen
      sandbox="allow-scripts allow-same-origin allow-popups"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  url: string
  title?: string
}>(), {
  title: ''
})

const embedUrl = computed(() => {
  if (!props.url) return ''

  // Bilibili: extract BV id
  const biliMatch = props.url.match(/bilibili\.com\/video\/(BV[\w]+)/)
  if (biliMatch) {
    return `//player.bilibili.com/player.html?bvid=${biliMatch[1]}&autoplay=0`
  }

  // YouTube: youtube.com/watch?v=xxx
  const ytMatch = props.url.match(/youtube\.com\/watch\?v=([\w-]+)/)
  if (ytMatch) {
    return `//www.youtube.com/embed/${ytMatch[1]}`
  }

  // YouTube: youtu.be/xxx
  const ytShortMatch = props.url.match(/youtu\.be\/([\w-]+)/)
  if (ytShortMatch) {
    return `//www.youtube.com/embed/${ytShortMatch[1]}`
  }

  return ''
})
</script>
