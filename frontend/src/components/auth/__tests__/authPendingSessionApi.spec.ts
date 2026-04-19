import { beforeEach, describe, expect, it, vi } from 'vitest'

const postMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    post: postMock
  }
}))

describe('auth pending session api contract', () => {
  beforeEach(() => {
    postMock.mockReset()
  })

  it('uses the create-account endpoint for linuxdo pending sessions even when invitation code is present', async () => {
    postMock.mockResolvedValue({
      data: {
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        expires_in: 3600,
        token_type: 'Bearer'
      }
    })

    const { createOAuthAccount } = await import('@/api/auth')

    await createOAuthAccount('linuxdo', {
      pendingAuthToken: 'pending-token',
      email: 'user@example.com',
      password: 'secret123',
      verifyCode: '123456',
      invitationCode: 'INVITE-001',
      adoptDisplayName: true,
      adoptAvatar: false
    })

    expect(postMock).toHaveBeenCalledTimes(1)
    expect(postMock).toHaveBeenCalledWith('/auth/oauth/linuxdo/create-account', {
      pending_auth_token: 'pending-token',
      pending_oauth_token: 'pending-token',
      email: 'user@example.com',
      password: 'secret123',
      verify_code: '123456',
      invitation_code: 'INVITE-001',
      adopt_display_name: true,
      adopt_avatar: false
    })
  })
})
