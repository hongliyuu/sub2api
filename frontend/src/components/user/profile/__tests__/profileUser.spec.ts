import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAppStore } from '@/stores/app'
import { resolveUserAvatarUrl, resolveUserBinding } from '@/components/user/profile/profileUser'

describe('profileUser avatar helpers', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('returns the stored avatar url when present', () => {
    expect(resolveUserAvatarUrl({
      avatar_url: 'data:image/png;base64,QUJD'
    } as any)).toBe('data:image/png;base64,QUJD')
  })

  it('returns null when the user has no avatar url', () => {
    expect(resolveUserAvatarUrl({
      avatar_url: '   '
    } as any)).toBeNull()
  })

  it('does not expose a wechat connect url when only mp login is enabled outside wechat', () => {
    const store = useAppStore()
    store.cachedPublicSettings = {
      linuxdo_oauth_enabled: true,
      oidc_oauth_enabled: true,
      wechat_login_open_enabled: false,
      wechat_login_mp_enabled: true
    } as any

    const binding = resolveUserBinding({
      account_bindings: {
        wechat: {
          provider: 'wechat',
          bound: false
        }
      }
    } as any, 'wechat')

    expect(binding.connectUrl).toBeNull()
  })

  it('does not treat a synthetic email as a bound local login fallback', () => {
    const binding = resolveUserBinding({
      email: 'wechat-union-abc@wechat-connect.invalid'
    } as any, 'email')

    expect(binding.bound).toBe(false)
    expect(binding.value).toBe('')
  })
})
