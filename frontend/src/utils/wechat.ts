/**
 * WeChat Pay Utilities
 * 微信支付相关工具函数
 */

import type { JSAPIPaymentParams } from '@/api/recharge'

// ==================== Types ====================

// WeixinJSBridge 接口定义
interface WeixinJSBridge {
  invoke: (
    method: string,
    params: Record<string, string>,
    callback: (res: { err_msg: string }) => void
  ) => void
}

// 声明全局 WeixinJSBridge 对象
declare global {
  interface Window {
    WeixinJSBridge?: WeixinJSBridge
  }
}

// 支付结果
export interface WeChatPayResult {
  success: boolean
  cancelled: boolean
  errorMessage?: string
}

// ==================== Environment Detection ====================

/**
 * 检测是否在微信浏览器环境中
 * 通过检测 userAgent 中是否包含 MicroMessenger
 */
export function isWeChatBrowser(): boolean {
  if (typeof navigator === 'undefined') {
    return false
  }
  const ua = navigator.userAgent.toLowerCase()
  return ua.indexOf('micromessenger') > -1
}

/**
 * 检测 WeixinJSBridge 是否可用
 */
export function isWeixinJSBridgeReady(): boolean {
  return typeof window !== 'undefined' && typeof window.WeixinJSBridge !== 'undefined'
}

// ==================== Payment Functions ====================

/**
 * 等待 WeixinJSBridge 就绪
 * 在微信浏览器中，WeixinJSBridge 可能在页面加载后才注入
 */
export function waitForWeixinJSBridge(timeout = 5000): Promise<void> {
  return new Promise((resolve, reject) => {
    if (isWeixinJSBridgeReady()) {
      resolve()
      return
    }

    // 设置超时
    const timeoutId = setTimeout(() => {
      reject(new Error('WeixinJSBridge 加载超时'))
    }, timeout)

    // 监听 WeixinJSBridgeReady 事件
    const handler = () => {
      clearTimeout(timeoutId)
      document.removeEventListener('WeixinJSBridgeReady', handler)
      resolve()
    }

    document.addEventListener('WeixinJSBridgeReady', handler)
  })
}

/**
 * 调用微信 JSAPI 支付
 * 使用 WeixinJSBridge.invoke('getBrandWCPayRequest', ...) 调起微信支付
 *
 * @param params JSAPI 支付参数（从后端获取）
 * @returns 支付结果
 */
export async function invokeWeChatPay(params: JSAPIPaymentParams): Promise<WeChatPayResult> {
  // 确保在微信环境中
  if (!isWeChatBrowser()) {
    return {
      success: false,
      cancelled: false,
      errorMessage: '请在微信中打开页面'
    }
  }

  // 等待 WeixinJSBridge 就绪
  try {
    await waitForWeixinJSBridge()
  } catch (error) {
    return {
      success: false,
      cancelled: false,
      errorMessage: '微信支付组件加载失败'
    }
  }

  // 调用微信支付
  return new Promise((resolve) => {
    // 再次检查 WeixinJSBridge 是否可用（防止边缘情况）
    if (!window.WeixinJSBridge) {
      resolve({
        success: false,
        cancelled: false,
        errorMessage: '微信支付组件不可用'
      })
      return
    }

    window.WeixinJSBridge.invoke(
      'getBrandWCPayRequest',
      {
        appId: params.appId,
        timeStamp: params.timeStamp,
        nonceStr: params.nonceStr,
        package: params.package,
        signType: params.signType,
        paySign: params.paySign
      },
      (res) => {
        // 微信支付返回结果
        // get_brand_wcpay_request:ok - 支付成功
        // get_brand_wcpay_request:cancel - 用户取消
        // get_brand_wcpay_request:fail - 支付失败
        const errMsg = res.err_msg

        if (errMsg === 'get_brand_wcpay_request:ok') {
          resolve({
            success: true,
            cancelled: false
          })
        } else if (errMsg === 'get_brand_wcpay_request:cancel') {
          resolve({
            success: false,
            cancelled: true,
            errorMessage: '支付已取消'
          })
        } else {
          resolve({
            success: false,
            cancelled: false,
            errorMessage: '支付失败，请重试'
          })
        }
      }
    )
  })
}
