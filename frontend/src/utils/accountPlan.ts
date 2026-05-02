import type { Account } from '@/types'

export type AccountPlanLevel =
  | 'free'
  | 'basic'
  | 'standard'
  | 'premium'
  | 'team'
  | 'enterprise'
  | 'abnormal'
  | 'unknown'

const PLAN_BADGE_CLASS: Record<AccountPlanLevel, string> = {
  free: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300',
  basic: 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300',
  standard: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
  premium: 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-300',
  team: 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300',
  enterprise: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
  abnormal: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300',
  unknown: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
}

const BASE_BADGE_CLASS = 'inline-flex items-center rounded-md px-2 py-1 text-xs font-medium'

const LABELS: Record<string, string> = {
  free: 'Free',
  trial: 'Trial',
  go: 'Go',
  plus: 'Plus',
  pro: 'Pro',
  max: 'Max',
  max_5x: 'Max 5x',
  max_20x: 'Max 20x',
  team: 'Team',
  business: 'Business',
  enterprise: 'Enterprise',
  google_one_free: 'Google One Free',
  google_ai_plus: 'Google AI Plus',
  google_ai_pro: 'Google AI Pro',
  google_ai_ultra: 'Google AI Ultra',
  ai_premium: 'Google AI Pro',
  google_one_unlimited: 'Google AI Ultra',
  aistudio_free: 'AI Studio Free',
  aistudio_paid: 'AI Studio Paid',
  gcp_standard: 'GCP Standard',
  gcp_enterprise: 'GCP Enterprise',
  standard: 'Standard',
  paid: 'Paid',
  payg: 'Pay-as-you-go',
  premium: 'Premium',
  ultra: 'Ultra',
  abnormal: 'Abnormal',
  unknown: 'Unknown'
}

const LEVEL_BY_VALUE: Record<string, AccountPlanLevel> = {
  free: 'free',
  trial: 'free',
  google_one_free: 'free',
  aistudio_free: 'free',
  go: 'basic',
  google_ai_plus: 'basic',
  plus: 'standard',
  pro: 'standard',
  paid: 'standard',
  payg: 'standard',
  aistudio_paid: 'standard',
  google_ai_pro: 'standard',
  ai_premium: 'standard',
  gcp_standard: 'standard',
  standard: 'standard',
  max: 'premium',
  max_5x: 'premium',
  max_20x: 'premium',
  ultra: 'premium',
  google_ai_ultra: 'premium',
  google_one_unlimited: 'premium',
  premium: 'premium',
  team: 'team',
  business: 'team',
  enterprise: 'enterprise',
  gcp_enterprise: 'enterprise',
  abnormal: 'abnormal',
  error: 'abnormal',
  unknown: 'unknown'
}

const normalizePlanValue = (value: unknown): string => {
  if (value === null || value === undefined) return ''
  return String(value).trim().toLowerCase().replace(/[\s-]+/g, '_')
}

const titleCasePlan = (value: string): string => {
  return value
    .replace(/[_-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/\b\w/g, (char) => char.toUpperCase())
}

const getCredentialString = (account: Account, key: string): string => {
  const credentials = account.credentials as Record<string, unknown> | undefined
  const value = credentials?.[key]
  return typeof value === 'string' ? value : ''
}

export const getAccountPlanValue = (account: Account): string => {
  const planType = normalizePlanValue(account.plan_type)
  if (planType) return planType

  if (account.platform === 'gemini' || account.platform === 'antigravity') {
    const tier = normalizePlanValue(getCredentialString(account, 'tier_id'))
    if (tier) return tier
  }

  const credentialPlanType = normalizePlanValue(getCredentialString(account, 'plan_type'))
  if (credentialPlanType) return credentialPlanType

  return ''
}

export const resolveAccountPlanLevel = (account: Account): AccountPlanLevel => {
  const value = getAccountPlanValue(account)
  if (!value) return 'unknown'
  if (LEVEL_BY_VALUE[value]) return LEVEL_BY_VALUE[value]

  if (value.includes('abnormal') || value.includes('error')) return 'abnormal'
  if (value.includes('enterprise')) return 'enterprise'
  if (value.includes('team') || value.includes('business')) return 'team'
  if (value.includes('max') || value.includes('ultra') || value.includes('opus') || value.includes('deep_think')) return 'premium'
  if (value.includes('paid') || value.includes('payg') || value.includes('plus') || value.includes('pro') || value.includes('standard')) return 'standard'
  if (value.includes('free') || value.includes('trial')) return 'free'

  return 'unknown'
}

export const getAccountPlanBadgeClass = (account: Account): string => {
  return `${BASE_BADGE_CLASS} ${PLAN_BADGE_CLASS[resolveAccountPlanLevel(account)]}`
}

export const getAccountPlanLabel = (account: Account): string => {
  const value = getAccountPlanValue(account)
  if (!value) return '-'
  return LABELS[value] || titleCasePlan(value)
}

export const hasAccountPlan = (account: Account): boolean => {
  return getAccountPlanValue(account) !== ''
}
