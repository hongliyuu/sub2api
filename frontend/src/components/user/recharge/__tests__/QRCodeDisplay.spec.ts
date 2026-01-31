import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import QRCodeDisplay from '../QRCodeDisplay.vue'

// Mock qrcode library
const mockToCanvas = vi.fn()
vi.mock('qrcode', () => ({
  default: {
    toCanvas: (...args: unknown[]) => mockToCanvas(...args)
  }
}))

// Mock vue-i18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('QRCodeDisplay.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockToCanvas.mockResolvedValue(undefined)
  })

  it('初始显示二维码区域当未开始生成', () => {
    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: ''
      }
    })

    // 没有 codeUrl 时，显示空的二维码区域（非 loading，非 error）
    expect(wrapper.find('[data-testid="qrcode-state"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="loading-state"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="error-state"]').exists()).toBe(false)
  })

  it('生成二维码当提供有效 codeUrl', async () => {
    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=abc123'
      }
    })

    await flushPromises()

    expect(mockToCanvas).toHaveBeenCalled()
    expect(mockToCanvas).toHaveBeenCalledWith(
      expect.any(HTMLCanvasElement),
      'weixin://wxpay/bizpayurl?pr=abc123',
      expect.objectContaining({
        width: 200,
        margin: 1
      })
    )
  })

  it('使用自定义尺寸', async () => {
    mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=abc123',
        size: 300
      }
    })

    await flushPromises()

    expect(mockToCanvas).toHaveBeenCalledWith(
      expect.any(HTMLCanvasElement),
      'weixin://wxpay/bizpayurl?pr=abc123',
      expect.objectContaining({
        width: 300
      })
    )
  })

  it('显示错误状态当生成失败', async () => {
    mockToCanvas.mockRejectedValueOnce(new Error('Generation failed'))

    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=invalid'
      }
    })

    await flushPromises()

    expect(wrapper.find('[data-testid="error-state"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('recharge.qrcodeError')
  })

  it('触发 generated 事件当成功生成', async () => {
    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=abc123'
      }
    })

    await flushPromises()

    expect(wrapper.emitted('generated')).toBeTruthy()
  })

  it('触发 error 事件当生成失败', async () => {
    const error = new Error('Generation failed')
    mockToCanvas.mockRejectedValueOnce(error)

    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=test'
      }
    })

    await flushPromises()

    expect(wrapper.emitted('error')).toBeTruthy()
  })

  it('当 codeUrl 变化时重新生成二维码', async () => {
    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=first'
      }
    })

    await flushPromises()
    expect(mockToCanvas).toHaveBeenCalledTimes(1)

    await wrapper.setProps({
      codeUrl: 'weixin://wxpay/bizpayurl?pr=second'
    })
    await flushPromises()

    expect(mockToCanvas).toHaveBeenCalledTimes(2)
  })

  it('点击重试按钮重新生成二维码', async () => {
    mockToCanvas
      .mockRejectedValueOnce(new Error('First attempt failed'))
      .mockResolvedValueOnce(undefined)

    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=abc123'
      }
    })

    await flushPromises()

    // 应该显示错误状态
    expect(wrapper.find('[data-testid="error-state"]').exists()).toBe(true)

    // 点击重试
    await wrapper.find('button').trigger('click')
    await flushPromises()

    // 应该成功生成（不再显示错误状态）
    expect(wrapper.find('[data-testid="error-state"]').exists()).toBe(false)
    expect(wrapper.emitted('generated')).toBeTruthy()
  })

  it('二维码默认尺寸为 200x200 像素', async () => {
    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=abc123'
      }
    })

    await flushPromises()

    const canvas = wrapper.find('[data-testid="qrcode-canvas"]')
    expect(canvas.exists()).toBe(true)
    expect(canvas.attributes('width')).toBe('200')
    expect(canvas.attributes('height')).toBe('200')
  })

  it('显示扫码提示文案', async () => {
    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=abc123'
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('recharge.qrcodeHint')
  })

  it('无效 URL 格式时显示错误状态', async () => {
    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'invalid-url-format'
      }
    })

    await flushPromises()

    expect(wrapper.find('[data-testid="error-state"]').exists()).toBe(true)
    expect(wrapper.emitted('error')).toBeTruthy()
    expect(mockToCanvas).not.toHaveBeenCalled()
  })

  it('重试次数达到上限后禁用重试按钮', async () => {
    mockToCanvas.mockRejectedValue(new Error('Always fails'))

    const wrapper = mount(QRCodeDisplay, {
      props: {
        codeUrl: 'weixin://wxpay/bizpayurl?pr=abc123'
      }
    })

    await flushPromises()
    expect(wrapper.find('[data-testid="error-state"]').exists()).toBe(true)

    // 点击重试 3 次
    for (let i = 0; i < 3; i++) {
      const button = wrapper.find('button')
      if (!button.attributes('disabled')) {
        await button.trigger('click')
        await flushPromises()
      }
    }

    // 第 4 次应该禁用
    const button = wrapper.find('button')
    expect(button.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('recharge.maxRetriesReached')
  })
})
