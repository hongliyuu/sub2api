import type {
  User,
  UserAccountBinding,
  UserAccountBindingProvider
} from '@/types'

export interface ResolvedUserBinding {
  provider: UserAccountBindingProvider
  bound: boolean
  value: string | null
  verified: boolean | null
  connectedAt: string | null
  canDisconnect: boolean
  connectUrl: string | null
  disconnectUrl: string | null
  managed: boolean
}

type LooseRecord = Record<string, unknown>

const bindingProviders: UserAccountBindingProvider[] = ['email', 'linuxdo', 'wechat']
const bindRedirect = encodeURIComponent('/profile?oauth_intent=bind')

function asRecord(value: unknown): LooseRecord | null {
  return value && typeof value === 'object' ? (value as LooseRecord) : null
}

function readString(value: unknown): string | null {
  if (typeof value !== 'string') {
    return null
  }

  const trimmed = value.trim()
  return trimmed ? trimmed : null
}

function readBoolean(value: unknown): boolean | null {
  return typeof value === 'boolean' ? value : null
}

function firstString(...values: unknown[]): string | null {
  for (const value of values) {
    const normalized = readString(value)
    if (normalized) {
      return normalized
    }
  }
  return null
}

function firstBoolean(...values: unknown[]): boolean | null {
  for (const value of values) {
    const normalized = readBoolean(value)
    if (normalized !== null) {
      return normalized
    }
  }
  return null
}

function findBindingEntry(
  user: Partial<User> | null | undefined,
  provider: UserAccountBindingProvider
): UserAccountBinding | null {
  if (!user?.account_bindings) {
    return null
  }

  if (Array.isArray(user.account_bindings)) {
    return (
      user.account_bindings.find((entry) => {
        const providerName = readString(entry?.provider)
        return providerName === provider
      }) ?? null
    )
  }

  return user.account_bindings[provider] ?? null
}

function normalizeAvatarUrl(value: string | null): string | null {
  if (!value) {
    return null
  }

  if (/^(data:|blob:|https?:\/\/)/i.test(value)) {
    return value
  }

  try {
    return new URL(value, window.location.origin).toString()
  } catch {
    return value
  }
}

function readAvatarObjectUrl(rawUser: LooseRecord | null | undefined): string | null {
  const avatar = asRecord(rawUser?.avatar)
  return firstString(avatar?.url, avatar?.avatar_url, avatar?.avatarUrl)
}

function readProviderAvatarUrl(rawUser: LooseRecord | null | undefined): string | null {
  const identities = rawUser?.external_identities
  if (!Array.isArray(identities)) {
    return null
  }

  for (const item of identities) {
    const identity = asRecord(item)
    const avatarUrl = firstString(identity?.avatar_url, identity?.avatarUrl)
    if (avatarUrl) {
      return avatarUrl
    }
  }

  return null
}

export function resolveUserAvatarUrl(user: Partial<User> | null | undefined): string | null {
  const rawUser = user as LooseRecord | null | undefined
  const profile = asRecord(rawUser?.profile)

  return normalizeAvatarUrl(
    firstString(
      readAvatarObjectUrl(rawUser),
      user?.avatar_url,
      user?.avatar_thumbnail_url,
      rawUser?.avatarUrl,
      rawUser?.avatar,
      profile?.avatar_url,
      profile?.avatarUrl,
      readProviderAvatarUrl(rawUser)
    )
  )
}

export function getUserInitials(user: Partial<User> | null | undefined): string {
  const source = (user?.username || user?.email || 'U').trim()
  if (!source) {
    return 'U'
  }

  const parts = source.split(/[\s@._-]+/).filter(Boolean)
  if (parts.length >= 2) {
    return `${parts[0][0] || ''}${parts[1][0] || ''}`.toUpperCase()
  }

  return source.slice(0, 2).toUpperCase()
}

export function getUserDisplayName(user: Partial<User> | null | undefined): string {
  if (!user) {
    return ''
  }

  return user.username?.trim() || user.email?.split('@')[0] || user.email || ''
}

export function formatBindingValue(provider: UserAccountBindingProvider, value: string | null): string | null {
  if (!value) {
    return null
  }

  if (provider === 'wechat' && value.length > 12) {
    return `${value.slice(0, 4)}...${value.slice(-4)}`
  }

  return value
}

export function resolveUserBinding(
  user: Partial<User> | null | undefined,
  provider: UserAccountBindingProvider
): ResolvedUserBinding {
  const entry = findBindingEntry(user, provider)
  const defaultConnectUrl = provider === 'email'
    ? null
    : `/api/v1/auth/oauth/${provider}/start?intent=bind&redirect=${bindRedirect}`

  if (provider === 'email') {
    const value = firstString(
      entry?.display_name,
      entry?.value,
      entry?.identifier,
      entry && (entry as LooseRecord).email,
      user?.email
    )

    return {
      provider,
      bound: firstBoolean(entry?.bound) ?? Boolean(value),
      value,
      verified: firstBoolean(entry?.verified, user?.email_verified),
      connectedAt: firstString(entry?.connected_at),
      canDisconnect: false,
      connectUrl: defaultConnectUrl,
      disconnectUrl: null,
      managed: Boolean(entry)
    }
  }

  if (provider === 'linuxdo') {
    const value = firstString(
      entry?.display_name,
      entry?.value,
      entry?.identifier,
      user?.linuxdo_username,
      user?.linuxdo_subject,
      user?.linuxdo_id
    )

    return {
      provider,
      bound: firstBoolean(entry?.bound, user?.linuxdo_bound) ?? Boolean(value),
      value,
      verified: firstBoolean(entry?.verified),
      connectedAt: firstString(entry?.connected_at),
      canDisconnect: firstBoolean(entry?.can_disconnect) ?? Boolean(entry),
      connectUrl: firstString(entry?.connect_url) ?? defaultConnectUrl,
      disconnectUrl: firstString(entry?.disconnect_url),
      managed: Boolean(entry)
    }
  }

  const value = firstString(
    entry?.display_name,
    entry?.value,
    entry?.identifier,
    user?.wechat_nickname,
    user?.wechat_openid,
    user?.wechat_unionid,
    user?.wechat
  )

  return {
    provider,
    bound: firstBoolean(entry?.bound, user?.wechat_bound) ?? Boolean(value),
    value,
    verified: firstBoolean(entry?.verified),
    connectedAt: firstString(entry?.connected_at),
    canDisconnect: firstBoolean(entry?.can_disconnect) ?? Boolean(entry),
    connectUrl: firstString(entry?.connect_url) ?? defaultConnectUrl,
    disconnectUrl: firstString(entry?.disconnect_url),
    managed: Boolean(entry)
  }
}

export function resolveUserBindings(
  user: Partial<User> | null | undefined
): ResolvedUserBinding[] {
  return bindingProviders.map((provider) => resolveUserBinding(user, provider))
}

export function supportsManagedBindings(user: Partial<User> | null | undefined): boolean {
  return resolveUserBindings(user).some((binding) => binding.managed)
}
