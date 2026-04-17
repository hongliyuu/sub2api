import { describe, expect, it } from 'vitest'
import {
  formatBindingValue,
  getUserInitials,
  resolveUserAvatarUrl,
  resolveUserBindings,
  supportsManagedBindings
} from '../profileUser'

describe('profileUser helpers', () => {
  it('prefers explicit avatar fields and normalizes relative URLs', () => {
    const originalWindow = globalThis.window
    Object.defineProperty(globalThis, 'window', {
      value: { location: { origin: 'https://example.com' } },
      configurable: true
    })

    expect(resolveUserAvatarUrl({ avatar_url: '/uploads/avatar.webp' })).toBe(
      'https://example.com/uploads/avatar.webp'
    )

    Object.defineProperty(globalThis, 'window', {
      value: originalWindow,
      configurable: true
    })
  })

  it('falls back to nested avatar payloads and external identity avatars', () => {
    const originalWindow = globalThis.window
    Object.defineProperty(globalThis, 'window', {
      value: { location: { origin: 'https://example.com' } },
      configurable: true
    })

    expect(resolveUserAvatarUrl({
      avatar: { url: '/uploads/avatar-from-nested.webp' }
    } as any)).toBe('https://example.com/uploads/avatar-from-nested.webp')

    expect(resolveUserAvatarUrl({
      external_identities: [
        { provider: 'wechat', avatar_url: '/uploads/avatar-from-provider.webp' }
      ]
    } as any)).toBe('https://example.com/uploads/avatar-from-provider.webp')

    Object.defineProperty(globalThis, 'window', {
      value: originalWindow,
      configurable: true
    })
  })

  it('extracts initials from username or email', () => {
    expect(getUserInitials({ username: 'Jane Doe' })).toBe('JD')
    expect(getUserInitials({ email: 'linuxdo_user@example.com' })).toBe('LU')
  })

  it('resolves managed bindings from record-based account bindings', () => {
    const bindings = resolveUserBindings({
      email: 'user@example.com',
      email_verified: true,
      account_bindings: {
        linuxdo: {
          provider: 'linuxdo',
          bound: true,
          value: 'linuxdo_shaw',
          can_disconnect: true
        },
        wechat: {
          provider: 'wechat',
          bound: false,
          connect_url: '/oauth/wechat/start'
        }
      }
    })

    expect(bindings.find((binding) => binding.provider === 'email')?.bound).toBe(true)
    expect(bindings.find((binding) => binding.provider === 'linuxdo')?.value).toBe('linuxdo_shaw')
    expect(bindings.find((binding) => binding.provider === 'wechat')?.managed).toBe(true)
    expect(supportsManagedBindings({
      account_bindings: {
        linuxdo: { provider: 'linuxdo', bound: false }
      }
    })).toBe(true)
  })

  it('falls back to top-level binding fields and masks long wechat ids', () => {
    const bindings = resolveUserBindings({
      email: 'user@example.com',
      linuxdo_username: 'shaw',
      wechat_openid: 'wx1234567890abcd'
    })

    expect(bindings.find((binding) => binding.provider === 'linuxdo')?.bound).toBe(true)
    expect(formatBindingValue('wechat', bindings.find((binding) => binding.provider === 'wechat')?.value || null)).toBe(
      'wx12...abcd'
    )
  })
})
