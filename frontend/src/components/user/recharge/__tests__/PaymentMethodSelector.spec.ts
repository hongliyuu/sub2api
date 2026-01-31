import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import PaymentMethodSelector from '../PaymentMethodSelector.vue'

// 创建 i18n 实例
const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: {
    zh: {
      recharge: {
        paymentMethod: '支付方式',
        selectPaymentMethod: '选择{method}支付',
        wechatPay: '微信支付',
        wechatPayDesc: '使用微信扫码支付',
        alipay: '支付宝',
        alipayDesc: '使用支付宝支付',
        comingSoonLabel: '即将上线'
      }
    }
  }
})

describe('PaymentMethodSelector', () => {
  const createWrapper = (props = {}) => {
    return mount(PaymentMethodSelector, {
      props: {
        modelValue: null,
        ...props
      },
      global: {
        plugins: [i18n]
      }
    })
  }

  describe('渲染', () => {
    it('应该渲染支付方式标签', () => {
      const wrapper = createWrapper()
      expect(wrapper.text()).toContain('支付方式')
    })

    it('应该渲染微信支付选项', () => {
      const wrapper = createWrapper()
      expect(wrapper.text()).toContain('微信支付')
      expect(wrapper.text()).toContain('使用微信扫码支付')
    })

    it('应该渲染支付宝选项（禁用状态）', () => {
      const wrapper = createWrapper()
      expect(wrapper.text()).toContain('支付宝')
      expect(wrapper.text()).toContain('即将上线')
    })

    it('应该渲染正确数量的支付选项', () => {
      const wrapper = createWrapper()
      const options = wrapper.findAll('[data-testid="payment-method-option"]')
      expect(options.length).toBe(2)
    })
  })

  describe('选择功能', () => {
    it('点击微信支付应该触发 update:modelValue 事件', async () => {
      const wrapper = createWrapper()
      const options = wrapper.findAll('[data-testid="payment-method-option"]')

      await options[0].trigger('click')

      expect(wrapper.emitted('update:modelValue')).toBeTruthy()
      expect(wrapper.emitted('update:modelValue')![0]).toEqual(['wechat_pay'])
    })

    it('点击禁用的支付宝选项不应该触发事件', async () => {
      const wrapper = createWrapper()
      const options = wrapper.findAll('[data-testid="payment-method-option"]')

      await options[1].trigger('click')

      expect(wrapper.emitted('update:modelValue')).toBeFalsy()
    })

    it('选中状态应该正确显示', () => {
      const wrapper = createWrapper({ modelValue: 'wechat_pay' })
      const options = wrapper.findAll('[data-testid="payment-method-option"]')

      expect(options[0].classes()).toContain('option-selected')
      expect(options[1].classes()).not.toContain('option-selected')
    })
  })

  describe('禁用状态', () => {
    it('禁用的选项应该有 disabled 属性', () => {
      const wrapper = createWrapper()
      const options = wrapper.findAll('[data-testid="payment-method-option"]')

      expect(options[0].attributes('disabled')).toBeUndefined()
      expect(options[1].attributes('disabled')).toBeDefined()
    })

    it('禁用的选项应该显示"即将上线"标签', () => {
      const wrapper = createWrapper()
      expect(wrapper.text()).toContain('即将上线')
    })
  })

  describe('无障碍', () => {
    it('每个选项应该有 aria-label', () => {
      const wrapper = createWrapper()
      const options = wrapper.findAll('[data-testid="payment-method-option"]')

      expect(options[0].attributes('aria-label')).toContain('微信支付')
      expect(options[1].attributes('aria-label')).toContain('支付宝')
    })

    it('所有选项都应该是 button 类型', () => {
      const wrapper = createWrapper()
      const options = wrapper.findAll('[data-testid="payment-method-option"]')

      options.forEach((option) => {
        expect(option.attributes('type')).toBe('button')
      })
    })
  })

  describe('样式状态', () => {
    it('未选中状态应该使用默认样式', () => {
      const wrapper = createWrapper({ modelValue: null })
      const options = wrapper.findAll('[data-testid="payment-method-option"]')

      expect(options[0].classes()).toContain('option-default')
    })

    it('选中状态应该使用选中样式', () => {
      const wrapper = createWrapper({ modelValue: 'wechat_pay' })
      const options = wrapper.findAll('[data-testid="payment-method-option"]')

      expect(options[0].classes()).toContain('option-selected')
    })
  })

  describe('图标渲染', () => {
    it('应该渲染微信支付图标', () => {
      const wrapper = createWrapper()
      const svgs = wrapper.findAll('svg')
      expect(svgs.length).toBeGreaterThan(0)
    })
  })
})
