import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { isWeChatBrowser, isWeixinJSBridgeReady, waitForWeixinJSBridge, invokeWeChatPay } from '@/utils/wechat'
import type { JSAPIPaymentParams } from '@/api/recharge'

describe('WeChat Pay Utilities', () => {
  const originalNavigator = global.navigator
  const originalWindow = global.window

  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
    // Restore original objects
    Object.defineProperty(global, 'navigator', {
      value: originalNavigator,
      writable: true,
      configurable: true
    })
    Object.defineProperty(global, 'window', {
      value: originalWindow,
      writable: true,
      configurable: true
    })
  })

  describe('isWeChatBrowser', () => {
    it('should return true when userAgent contains MicroMessenger', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) MicroMessenger/8.0.0'
        },
        writable: true,
        configurable: true
      })

      expect(isWeChatBrowser()).toBe(true)
    })

    it('should return true when userAgent contains micromessenger (lowercase)', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 micromessenger/8.0.0'
        },
        writable: true,
        configurable: true
      })

      expect(isWeChatBrowser()).toBe(true)
    })

    it('should return false when userAgent does not contain MicroMessenger', () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) Safari/605.1.15'
        },
        writable: true,
        configurable: true
      })

      expect(isWeChatBrowser()).toBe(false)
    })

    it('should return false when navigator is undefined', () => {
      // @ts-expect-error testing undefined navigator
      delete global.navigator

      expect(isWeChatBrowser()).toBe(false)
    })
  })

  describe('isWeixinJSBridgeReady', () => {
    it('should return true when WeixinJSBridge is defined', () => {
      Object.defineProperty(global, 'window', {
        value: {
          WeixinJSBridge: {
            invoke: vi.fn()
          }
        },
        writable: true,
        configurable: true
      })

      expect(isWeixinJSBridgeReady()).toBe(true)
    })

    it('should return false when WeixinJSBridge is undefined', () => {
      Object.defineProperty(global, 'window', {
        value: {},
        writable: true,
        configurable: true
      })

      expect(isWeixinJSBridgeReady()).toBe(false)
    })

    it('should return false when window is undefined', () => {
      // @ts-expect-error testing undefined window
      delete global.window

      expect(isWeixinJSBridgeReady()).toBe(false)
    })
  })

  describe('waitForWeixinJSBridge', () => {
    it('should resolve immediately if WeixinJSBridge is ready', async () => {
      Object.defineProperty(global, 'window', {
        value: {
          WeixinJSBridge: {
            invoke: vi.fn()
          }
        },
        writable: true,
        configurable: true
      })

      await expect(waitForWeixinJSBridge()).resolves.toBeUndefined()
    })

    it('should reject after timeout if WeixinJSBridge is not available', async () => {
      Object.defineProperty(global, 'window', {
        value: {},
        writable: true,
        configurable: true
      })

      const promise = waitForWeixinJSBridge(1000)
      vi.advanceTimersByTime(1000)

      await expect(promise).rejects.toThrow('WeixinJSBridge 加载超时')
    })

    it('should resolve when WeixinJSBridgeReady event is dispatched', async () => {
      const mockDocument = {
        addEventListener: vi.fn(),
        removeEventListener: vi.fn()
      }
      Object.defineProperty(global, 'document', {
        value: mockDocument,
        writable: true,
        configurable: true
      })
      Object.defineProperty(global, 'window', {
        value: {},
        writable: true,
        configurable: true
      })

      const promise = waitForWeixinJSBridge(5000)

      // Simulate the event being triggered
      const eventCallback = mockDocument.addEventListener.mock.calls[0][1]
      eventCallback()

      await expect(promise).resolves.toBeUndefined()
      expect(mockDocument.removeEventListener).toHaveBeenCalled()
    })
  })

  describe('invokeWeChatPay', () => {
    const mockParams: JSAPIPaymentParams = {
      appId: 'wx12345',
      timeStamp: '1234567890',
      nonceStr: 'abc123',
      package: 'prepay_id=test123',
      signType: 'RSA',
      paySign: 'sign123'
    }

    it('should return error if not in WeChat browser', async () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'Mozilla/5.0 Safari/605.1.15'
        },
        writable: true,
        configurable: true
      })

      const result = await invokeWeChatPay(mockParams)

      expect(result.success).toBe(false)
      expect(result.cancelled).toBe(false)
      expect(result.errorMessage).toBe('请在微信中打开页面')
    })

    it('should return success when payment succeeds', async () => {
      // Setup WeChat environment
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'MicroMessenger/8.0.0'
        },
        writable: true,
        configurable: true
      })

      const mockInvoke = vi.fn((method, params, callback) => {
        callback({ err_msg: 'get_brand_wcpay_request:ok' })
      })

      Object.defineProperty(global, 'window', {
        value: {
          WeixinJSBridge: {
            invoke: mockInvoke
          }
        },
        writable: true,
        configurable: true
      })

      const result = await invokeWeChatPay(mockParams)

      expect(result.success).toBe(true)
      expect(result.cancelled).toBe(false)
      expect(mockInvoke).toHaveBeenCalledWith(
        'getBrandWCPayRequest',
        {
          appId: mockParams.appId,
          timeStamp: mockParams.timeStamp,
          nonceStr: mockParams.nonceStr,
          package: mockParams.package,
          signType: mockParams.signType,
          paySign: mockParams.paySign
        },
        expect.any(Function)
      )
    })

    it('should return cancelled when user cancels payment', async () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'MicroMessenger/8.0.0'
        },
        writable: true,
        configurable: true
      })

      const mockInvoke = vi.fn((method, params, callback) => {
        callback({ err_msg: 'get_brand_wcpay_request:cancel' })
      })

      Object.defineProperty(global, 'window', {
        value: {
          WeixinJSBridge: {
            invoke: mockInvoke
          }
        },
        writable: true,
        configurable: true
      })

      const result = await invokeWeChatPay(mockParams)

      expect(result.success).toBe(false)
      expect(result.cancelled).toBe(true)
      expect(result.errorMessage).toBe('支付已取消')
    })

    it('should return failure when payment fails', async () => {
      Object.defineProperty(global, 'navigator', {
        value: {
          userAgent: 'MicroMessenger/8.0.0'
        },
        writable: true,
        configurable: true
      })

      const mockInvoke = vi.fn((method, params, callback) => {
        callback({ err_msg: 'get_brand_wcpay_request:fail' })
      })

      Object.defineProperty(global, 'window', {
        value: {
          WeixinJSBridge: {
            invoke: mockInvoke
          }
        },
        writable: true,
        configurable: true
      })

      const result = await invokeWeChatPay(mockParams)

      expect(result.success).toBe(false)
      expect(result.cancelled).toBe(false)
      expect(result.errorMessage).toBe('支付失败，请重试')
    })
  })
})
