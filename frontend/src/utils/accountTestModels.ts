import type { Account, ClaudeModel } from '@/types'

const prioritizedGeminiModels = [
  'gemini-3.1-flash-image-preview',
  'gemini-2.5-flash-image',
  'gemini-2.5-flash',
  'gemini-2.5-pro',
  'gemini-3-flash',
  'gemini-3-pro-preview',
  'gemini-2.0-flash'
]

const prioritizedVertexModels = [
  'gemini-3-flash-preview',
  'gemini-3-pro-preview',
  'gemini-3.1-pro-preview',
  'gemini-3.1-flash-image-preview',
  'gemini-3.1-pro-image-preview',
  'gemini-3.1-flash-lite-preview'
]

const prioritizedAntigravityModels = [
  'gemini-3.1-flash-image-preview',
  'gemini-3.1-pro-preview',
  'gemini-3-flash',
  'gemini-3-pro',
  'gemini-3.1-flash-preview'
]

function getPriorityOrder(account: Pick<Account, 'platform' | 'type'> | null | undefined): string[] {
  if (account?.type === 'vertex') return prioritizedVertexModels
  if (account?.platform === 'antigravity') return prioritizedAntigravityModels
  return prioritizedGeminiModels
}

export function sortAccountTestModels(
  models: ClaudeModel[],
  account: Pick<Account, 'platform' | 'type'> | null | undefined
): ClaudeModel[] {
  const priorityMap = new Map(getPriorityOrder(account).map((id, index) => [id, index]))

  return [...models].sort((a, b) => {
    const aPriority = priorityMap.get(a.id) ?? Number.MAX_SAFE_INTEGER
    const bPriority = priorityMap.get(b.id) ?? Number.MAX_SAFE_INTEGER
    if (aPriority !== bPriority) return aPriority - bPriority
    return 0
  })
}

export function defaultAccountTestModelId(
  models: ClaudeModel[],
  account: Pick<Account, 'platform' | 'type'> | null | undefined
): string {
  if (models.length === 0) return ''

  if (account?.platform === 'gemini') {
    return models[0].id
  }

  const sonnetModel = models.find((model) => model.id.includes('sonnet'))
  return sonnetModel?.id || models[0].id
}
