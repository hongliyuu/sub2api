import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ThirdPartyAuthCallbackFlow from '@/components/auth/ThirdPartyAuthCallbackFlow.vue'
import LinuxDoCallbackView from '@/views/auth/LinuxDoCallbackView.vue'
import OidcCallbackView from '@/views/auth/OidcCallbackView.vue'
import WechatCallbackView from '@/views/auth/WechatCallbackView.vue'

const {
  replaceMock,
  appStoreMock,
  createOAuthAccountMock,
  persistOAuthTokenPairMock,
  getPublicSettingsMock,
  bindAccountMock,
  setAccountBindingAdoptionDecisionMock
} = vi.hoisted(() => ({
  replaceMock: vi.fn(),
  appStoreMock: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
    siteName: 'Sub2API',
    thirdPartyFirstLoginRequireEmail: false,
    cachedPublicSettings: null as null | Record<string, unknown>,
    fetchPublicSettings: vi.fn()
  },
  createOAuthAccountMock: vi.fn(),
  persistOAuthTokenPairMock: vi.fn(),
  getPublicSettingsMock: vi.fn(),
  bindAccountMock: vi.fn(),
  setAccountBindingAdoptionDecisionMock: vi.fn()
}))

const authStore = {
  token: null as string | null,
  pendingAuthSession: null as Record<string, unknown> | null,
  setToken: vi.fn(),
  setPendingAuthSession: vi.fn(),
  clearPendingAuthSession: vi.fn(),
  refreshUser: vi.fn()
}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    replace: replaceMock
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => authStore,
  useAppStore: () => appStoreMock
}))

vi.mock('@/api/auth', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/auth')>()
  return {
    ...actual,
    createOAuthAccount: createOAuthAccountMock,
    persistOAuthTokenPair: persistOAuthTokenPairMock,
    getPublicSettings: getPublicSettingsMock
  }
})

vi.mock('@/api/user', () => ({
  userAPI: {
    bindAccount: bindAccountMock,
    setAccountBindingAdoptionDecision: setAccountBindingAdoptionDecisionMock
  }
}))

type CallbackComponent = typeof LinuxDoCallbackView | typeof OidcCallbackView | typeof WechatCallbackView

function mountView(component: CallbackComponent) {
  return mount(component, {
    global: {
      stubs: {
        AuthLayout: {
          template: '<div><slot /></div>'
        }
      }
    }
  })
}

function setColdStartSettings(settings: Record<string, unknown>) {
  appStoreMock.thirdPartyFirstLoginRequireEmail = false
  appStoreMock.cachedPublicSettings = null
  appStoreMock.fetchPublicSettings.mockImplementation(async () => {
    appStoreMock.thirdPartyFirstLoginRequireEmail =
      settings.third_party_first_login_require_email === true
    appStoreMock.cachedPublicSettings = settings
    return settings
  })
  getPublicSettingsMock.mockResolvedValue({
    oidc_oauth_provider_name: 'OIDC',
    ...settings
  })
}

describe.each([
  ['LinuxDoCallbackView', LinuxDoCallbackView, 'linuxdo'],
  ['OidcCallbackView', OidcCallbackView, 'oidc'],
  ['WechatCallbackView', WechatCallbackView, 'wechat']
] as const)('%s cold-start public settings gating', (_name, component, provider) => {
  beforeEach(() => {
    replaceMock.mockReset()
    appStoreMock.showError.mockReset()
    appStoreMock.showSuccess.mockReset()
    appStoreMock.fetchPublicSettings.mockReset()
    createOAuthAccountMock.mockReset()
    persistOAuthTokenPairMock.mockReset()
    getPublicSettingsMock.mockReset()
    bindAccountMock.mockReset()
    setAccountBindingAdoptionDecisionMock.mockReset()
    authStore.token = null
    authStore.pendingAuthSession = null
  })

  it('routes pending create-account to Register when first-login email binding is required on a cold start', async () => {
    setColdStartSettings({
      third_party_first_login_require_email: true,
      invitation_code_enabled: false
    })
    const summary = {
      authResult: 'pending_session' as const,
      pendingAuthToken: 'pending-token',
      provider,
      intent: 'login' as const,
      redirect: '/welcome',
      adoptionRequired: false,
      suggestedDisplayName: null,
      suggestedAvatarUrl: null
    }

    const wrapper = mountView(component)
    await flushPromises()

    wrapper.findComponent(ThirdPartyAuthCallbackFlow).vm.$emit('create-account', summary)
    await flushPromises()

    expect(replaceMock).toHaveBeenCalledWith({
      name: 'Register',
      query: { redirect: '/welcome' }
    })
    expect(createOAuthAccountMock).not.toHaveBeenCalled()
  })

  it('routes pending create-account to Register when invitation code gating is only known after cold-start settings load', async () => {
    setColdStartSettings({
      third_party_first_login_require_email: false,
      invitation_code_enabled: true
    })
    const summary = {
      authResult: 'pending_session' as const,
      pendingAuthToken: 'pending-token',
      provider,
      intent: 'login' as const,
      redirect: '/welcome',
      adoptionRequired: false,
      suggestedDisplayName: null,
      suggestedAvatarUrl: null
    }

    const wrapper = mountView(component)
    await flushPromises()

    wrapper.findComponent(ThirdPartyAuthCallbackFlow).vm.$emit('create-account', summary)
    await flushPromises()

    expect(replaceMock).toHaveBeenCalledWith({
      name: 'Register',
      query: { redirect: '/welcome' }
    })
    expect(createOAuthAccountMock).not.toHaveBeenCalled()
  })
})
