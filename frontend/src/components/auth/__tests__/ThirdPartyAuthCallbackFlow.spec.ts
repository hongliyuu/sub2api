import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ThirdPartyAuthCallbackFlow from '../ThirdPartyAuthCallbackFlow.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('ThirdPartyAuthCallbackFlow', () => {
  it('handles auth_result=pending_session without requiring access_token', async () => {
    const wrapper = mount(ThirdPartyAuthCallbackFlow, {
      props: {
        hash:
          '#auth_result=pending_session&pending_auth_token=token-1&provider=oidc&intent=login&redirect=%2Fprofile'
      }
    })

    expect(wrapper.text()).toContain('auth.thirdParty.callback.pending.login.actions.bindExisting')
    expect(wrapper.emitted('pending-session')?.[0]?.[0]).toMatchObject({
      authResult: 'pending_session',
      pendingAuthToken: 'token-1',
      provider: 'oidc',
      intent: 'login',
      redirect: '/profile'
    })
  })

  it('renders bind_current_user state from the callback contract', () => {
    const wrapper = mount(ThirdPartyAuthCallbackFlow, {
      props: {
        hash:
          '#auth_result=pending_session&pending_auth_token=token-2&provider=wechat&intent=bind_current_user'
      }
    })

    expect(wrapper.text()).toContain('auth.thirdParty.callback.pending.bindCurrent.title')
    expect(wrapper.find('[data-testid="bind-current-user-action"]').exists()).toBe(true)
  })

  it('renders adopt_existing_user_by_email state from the callback contract', () => {
    const wrapper = mount(ThirdPartyAuthCallbackFlow, {
      props: {
        hash:
          '#auth_result=pending_session&pending_auth_token=token-3&provider=linuxdo&intent=adopt_existing_user_by_email'
      }
    })

    expect(wrapper.text()).toContain('auth.thirdParty.callback.pending.adoptExisting.title')
    expect(wrapper.find('[data-testid="adopt-existing-user-action"]').exists()).toBe(true)
  })

  it('opens adoption dialog and does not use legacy window.confirm prompts', () => {
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    const wrapper = mount(ThirdPartyAuthCallbackFlow, {
      props: {
        hash:
          '#access_token=access-1&refresh_token=refresh-1&expires_in=3600&provider=oidc&adoption_required=true&suggested_display_name=Acme%20User&suggested_avatar_url=https%3A%2F%2Fexample.com%2Favatar.png'
      }
    })

    expect(wrapper.text()).toContain('auth.thirdParty.callback.success.title')
    expect(wrapper.text()).toContain('Use provider nickname')
    expect(confirmSpy).not.toHaveBeenCalled()
  })
})
