<template>
  <div class="relative">
    <!-- 加载中 -->
    <div v-if="loading" class="flex h-[300px] items-center justify-center">
      <LoadingSpinner />
    </div>
    <!-- 未选中用户 -->
    <div v-else-if="!userId" class="flex h-[300px] items-center justify-center text-sm text-gray-400">
      请选择一个用户
    </div>
    <!-- 无数据 -->
    <div v-else-if="!dailyData" class="flex h-[300px] items-center justify-center text-sm text-gray-400">
      暂无数据
    </div>
    <!-- 选中用户不在当前区间 -->
    <div v-else-if="!userHasData" class="flex h-[300px] items-center justify-center text-sm text-gray-400">
      该用户在当前时间区间内无请求记录
    </div>
    <!-- 图表 -->
    <div v-else class="h-[300px]">
      <canvas ref="chartRef" class="!h-full !w-full" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import {
  Chart,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'
import type { CopilotUsersDailyStatsResult } from '@/api/admin/copilotAnalytics'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'

Chart.register(LineController, LineElement, PointElement, LinearScale, CategoryScale, Tooltip, Legend, Filler)

const props = withDefaults(defineProps<{
  days?: number
  userId: number | null
  dailyData: CopilotUsersDailyStatsResult | null
  loading?: boolean
}>(), {
  days: 30,
  loading: false,
})

/** 当前选中用户在 dailyData 中是否有数据 */
const userHasData = computed(() => {
  if (!props.userId || !props.dailyData) return false
  return props.dailyData.days.some(e => e.user_id === props.userId)
})

const chartRef = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null

/** 生成最近 N 天的日期字符串数组（本地时区，无 UTC 偏移） */
function buildDateRange(days: number): string[] {
  const dates: string[] = []
  const today = new Date()
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(today)
    d.setDate(d.getDate() - i)
    dates.push(
      `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`,
    )
  }
  return dates
}

function buildChart() {
  if (!props.userId || !props.dailyData || !chartRef.value || !userHasData.value) return

  const dates = buildDateRange(props.days ?? 30)

  // 按日期汇总该用户的 premium / agent
  const premiumByDate = new Map<string, number>()
  const agentByDate = new Map<string, number>()

  for (const entry of props.dailyData.days) {
    if (entry.user_id !== props.userId) continue
    premiumByDate.set(entry.date, (premiumByDate.get(entry.date) ?? 0) + entry.premium_count)
    agentByDate.set(entry.date, (agentByDate.get(entry.date) ?? 0) + entry.agent_count)
  }

  const premiumData = dates.map(d => premiumByDate.get(d) ?? 0)
  const agentData = dates.map(d => agentByDate.get(d) ?? 0)
  const totalData = dates.map((_, i) => premiumData[i] + agentData[i])

  const dark = document.documentElement.classList.contains('dark')
  const tickColor = dark ? '#9ca3af' : '#6b7280'
  const gridColor = dark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.06)'
  const legendColor = dark ? '#d1d5db' : '#374151'

  const pointRadius = (props.days ?? 30) <= 14 ? 3 : 0

  chart?.destroy()
  chart = new Chart(chartRef.value, {
    type: 'line',
    data: {
      labels: dates,
      datasets: [
        {
          label: 'Premium',
          data: premiumData,
          borderColor: '#10b981',
          backgroundColor: '#10b98118',
          borderWidth: 2,
          pointRadius,
          pointHoverRadius: 5,
          tension: 0.3,
          fill: false,
        },
        {
          label: 'Agent',
          data: agentData,
          borderColor: '#3b82f6',
          backgroundColor: '#3b82f618',
          borderWidth: 2,
          pointRadius,
          pointHoverRadius: 5,
          tension: 0.3,
          fill: false,
        },
        {
          label: '总量',
          data: totalData,
          borderColor: '#8b5cf6',
          backgroundColor: '#8b5cf618',
          borderWidth: 2,
          pointRadius,
          pointHoverRadius: 5,
          tension: 0.3,
          fill: false,
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: {
          position: 'bottom',
          labels: { boxWidth: 12, padding: 16, font: { size: 12 }, color: legendColor },
        },
        tooltip: {
          callbacks: {
            title: (items) => items[0]?.label ?? '',
            label: (item) => ` ${item.dataset.label}: ${item.parsed.y} 次`,
          },
        },
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: { maxTicksLimit: 10, font: { size: 11 }, color: tickColor },
        },
        y: {
          beginAtZero: true,
          grid: { color: gridColor },
          ticks: { font: { size: 11 }, color: tickColor },
        },
      },
    },
  })
}

async function rebuild() {
  await nextTick()
  buildChart()
}

onMounted(rebuild)
watch(() => [props.userId, props.dailyData, props.days, props.loading], rebuild)
onBeforeUnmount(() => chart?.destroy())
</script>
