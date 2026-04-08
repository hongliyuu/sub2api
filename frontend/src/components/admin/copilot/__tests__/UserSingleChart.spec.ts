import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'

import type { CopilotUsersDailyStatsResult } from '@/api/admin/copilotAnalytics'

// ── Mock Chart.js before importing the component ──────────────────────────────
vi.mock('chart.js', () => {
  const Chart = vi.fn().mockImplementation(() => ({
    data: {},
    update: vi.fn(),
    destroy: vi.fn(),
  }))
  ;(Chart as any).register = vi.fn()
  return {
    Chart,
    LineController: {},
    LineElement: {},
    PointElement: {},
    LinearScale: {},
    CategoryScale: {},
    Tooltip: {},
    Legend: {},
    Filler: {},
  }
})

// ── Mock LoadingSpinner（依赖 vue-i18n，测试环境无法直接挂载）────────────────
vi.mock('@/components/common/LoadingSpinner.vue', () => ({
  default: { template: '<div class="loading-spinner-stub" />' },
}))

// ── Mock data ─────────────────────────────────────────────────────────────────

const MOCK_DAILY_DATA: CopilotUsersDailyStatsResult = {
  users: [
    { user_id: 1, username: 'alice' },
    { user_id: 2, username: 'bob' },
  ],
  days: [
    { user_id: 1, date: '2026-04-07', premium_count: 10, agent_count: 3 },
    { user_id: 1, date: '2026-04-08', premium_count: 5, agent_count: 2 },
    { user_id: 2, date: '2026-04-07', premium_count: 8, agent_count: 1 },
  ],
}

// Lazily import after mocks are registered
const { default: UserSingleChart } = await import('../UserSingleChart.vue')

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('UserSingleChart', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('当 userId 为 null 时显示提示文案', () => {
    const wrapper = mount(UserSingleChart, {
      props: { userId: null, dailyData: MOCK_DAILY_DATA },
    })
    expect(wrapper.text()).toContain('请选择一个用户')
  })

  it('当 dailyData 为 null 时显示暂无数据', () => {
    const wrapper = mount(UserSingleChart, {
      props: { userId: 1, dailyData: null },
    })
    expect(wrapper.text()).toContain('暂无数据')
  })

  it('当 loading=true 时显示 spinner 而非暂无数据', () => {
    const wrapper = mount(UserSingleChart, {
      props: { userId: 1, dailyData: null, loading: true },
    })
    expect(wrapper.find('.loading-spinner-stub').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('暂无数据')
  })

  it('当选中用户不在 dailyData 中时显示区间提示', () => {
    const dataWithoutUser3: CopilotUsersDailyStatsResult = {
      users: [{ user_id: 1, username: 'alice' }],
      days: [{ user_id: 1, date: '2026-04-07', premium_count: 5, agent_count: 0 }],
    }
    const wrapper = mount(UserSingleChart, {
      props: { userId: 3, dailyData: dataWithoutUser3 },
    })
    expect(wrapper.text()).toContain('该用户在当前时间区间内无请求记录')
  })

  it('当 userId 和 dailyData 都有效时渲染 canvas', () => {
    const wrapper = mount(UserSingleChart, {
      props: { userId: 1, dailyData: MOCK_DAILY_DATA },
      attachTo: document.body,
    })
    expect(wrapper.find('canvas').exists()).toBe(true)
  })
})

