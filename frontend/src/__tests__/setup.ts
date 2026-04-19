/**
 * Vitest 测试环境设置
 * 提供全局 mock 和测试工具
 */
import { config } from '@vue/test-utils'
import { vi } from 'vitest'

const localStorageStore = new Map<string, string>()

const localStorageMock = {
  get length() {
    return localStorageStore.size
  },
  clear: vi.fn(() => {
    localStorageStore.clear()
  }),
  getItem: vi.fn((key: string) => localStorageStore.get(key) ?? null),
  key: vi.fn((index: number) => Array.from(localStorageStore.keys())[index] ?? null),
  removeItem: vi.fn((key: string) => {
    localStorageStore.delete(key)
  }),
  setItem: vi.fn((key: string, value: string) => {
    localStorageStore.set(key, String(value))
  }),
}

vi.stubGlobal('localStorage', localStorageMock)

// Mock requestIdleCallback (Safari < 15 不支持)
if (typeof globalThis.requestIdleCallback === 'undefined') {
  globalThis.requestIdleCallback = ((callback: IdleRequestCallback) => {
    return window.setTimeout(() => callback({ didTimeout: false, timeRemaining: () => 50 }), 1)
  }) as unknown as typeof requestIdleCallback
}

if (typeof globalThis.cancelIdleCallback === 'undefined') {
  globalThis.cancelIdleCallback = ((id: number) => {
    window.clearTimeout(id)
  }) as unknown as typeof cancelIdleCallback
}

// Mock IntersectionObserver
class MockIntersectionObserver {
  observe = vi.fn()
  disconnect = vi.fn()
  unobserve = vi.fn()
}

globalThis.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver

// Mock ResizeObserver
class MockResizeObserver {
  observe = vi.fn()
  disconnect = vi.fn()
  unobserve = vi.fn()
}

globalThis.ResizeObserver = MockResizeObserver as unknown as typeof ResizeObserver

// Vue Test Utils 全局配置
config.global.stubs = {
  // 可以在这里添加全局 stub
}

// 设置全局测试超时
vi.setConfig({ testTimeout: 10000 })
