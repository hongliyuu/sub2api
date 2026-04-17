import { describe, expect, it } from 'vitest'
import {
  resolveUserAvatarUrl,
  resolveUserBinding
} from '../profileUser'

describe('profileUser contract helpers', () => {
  it('prefers nested custom avatar and falls back to provider avatar after removal', () => {
    expect(resolveUserAvatarUrl({
      avatar: {
        url: 'https://cdn.example.com/custom.webp'
      },
      avatar_url: 'https://cdn.example.com/provider.png',
      external_identities: [
        { provider: 'wechat', avatar_url: 'https://cdn.example.com/provider.png' }
      ]
    } as any)).toBe('https://cdn.example.com/custom.webp')

    expect(resolveUserAvatarUrl({
      avatar_url: null,
      external_identities: [
        { provider: 'wechat', avatar_url: 'https://cdn.example.com/provider.png' }
      ]
    } as any)).toBe('https://cdn.example.com/provider.png')
  })

  it('defaults managed bind entry redirects with explicit bind intent', () => {
    const binding = resolveUserBinding({
      account_bindings: {
        wechat: {
          provider: 'wechat',
          bound: false
        }
      }
    }, 'wechat')

    expect(binding.connectUrl).toBe('/api/v1/auth/oauth/wechat/start?intent=bind&redirect=%2Fprofile%3Foauth_intent%3Dbind')
  })
})
