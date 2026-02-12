import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory, type Router } from 'vue-router'
import { nextTick } from 'vue'
import SettingsView from '@/views/admin/SettingsView.vue'

const mocks = vi.hoisted(() => ({
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  fetchPublicSettings: vi.fn()
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getSettings: mocks.getSettings,
      updateSettings: mocks.updateSettings
    }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mocks.showError,
    showSuccess: mocks.showSuccess,
    fetchPublicSettings: mocks.fetchPublicSettings
  })
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayoutStub',
    template: '<div data-test="app-layout"><slot /></div>'
  }
}))

vi.mock('@/views/admin/settings/SettingsGeneralTab.vue', () => ({
  default: {
    name: 'SettingsGeneralTab',
    props: ['form'],
    data() {
      return {
        draft: ''
      }
    },
    template:
      '<div data-test="general-tab"><input data-test="general-input" v-model="draft" /></div>'
  }
}))

vi.mock('@/views/admin/settings/SettingsAuthTab.vue', () => ({
  default: {
    name: 'SettingsAuthTab',
    props: ['form'],
    template: '<div data-test="auth-tab">auth</div>'
  }
}))

vi.mock('@/views/admin/settings/SettingsPaymentTab.vue', () => ({
  default: {
    name: 'SettingsPaymentTab',
    props: ['form'],
    template: '<div data-test="payment-tab">payment</div>'
  }
}))

vi.mock('@/views/admin/settings/SettingsEmailTab.vue', () => ({
  default: {
    name: 'SettingsEmailTab',
    props: ['form'],
    template: '<div data-test="email-tab">email</div>'
  }
}))

vi.mock('@/views/admin/settings/SettingsAdvancedTab.vue', () => ({
  default: {
    name: 'SettingsAdvancedTab',
    template: '<div data-test="advanced-tab">advanced</div>'
  }
}))

function createTestRouter(): Router {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/admin/settings',
        component: SettingsView
      }
    ]
  })
}

async function mountWithRoute(path: string) {
  const router = createTestRouter()
  await router.push(path)
  await router.isReady()

  const wrapper = mount(SettingsView, {
    global: {
      plugins: [router]
    }
  })

  await flushPromises()
  await nextTick()

  return { wrapper, router }
}

async function clickTab(wrapper: ReturnType<typeof mount>, tabLabelKey: string) {
  const tabButton = wrapper.findAll('button').find((button) => button.text() === tabLabelKey)
  expect(tabButton).toBeDefined()
  await tabButton!.trigger('click')
  await flushPromises()
  await nextTick()
}

describe('SettingsView tab query behavior', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getSettings.mockResolvedValue({})
    mocks.updateSettings.mockResolvedValue({})
    mocks.fetchPublicSettings.mockResolvedValue(undefined)
  })

  it('should normalize invalid tab query to general and update URL', async () => {
    const { wrapper, router } = await mountWithRoute('/admin/settings?tab=invalid_tab')

    expect(router.currentRoute.value.query.tab).toBe('general')
    expect(wrapper.find('[data-test="general-tab"]').exists()).toBe(true)
  })

  it('should sync tab UI when route query tab changes externally', async () => {
    const { wrapper, router } = await mountWithRoute('/admin/settings?tab=general')

    await router.push('/admin/settings?tab=auth')
    await flushPromises()
    await nextTick()

    expect(wrapper.find('[data-test="auth-tab"]').exists()).toBe(true)
  })

  it('should keep general tab local state after tab switch', async () => {
    const { wrapper } = await mountWithRoute('/admin/settings?tab=general')

    const input = wrapper.get('[data-test="general-input"]')
    await input.setValue('unsaved draft value')

    await clickTab(wrapper, 'admin.settings.tabs.auth')
    await clickTab(wrapper, 'admin.settings.tabs.general')

    expect((wrapper.get('[data-test="general-input"]').element as HTMLInputElement).value).toBe(
      'unsaved draft value'
    )
  })
})
