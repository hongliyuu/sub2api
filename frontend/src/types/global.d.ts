import type { PublicSettings } from '@/types'

declare global {
  interface WeixinJSBridge {
    invoke(
      method: 'getBrandWCPayRequest',
      params: Record<string, unknown>,
      callback: (response: Record<string, unknown>) => void
    ): void
  }

  interface Window {
    __APP_CONFIG__?: PublicSettings
    WeixinJSBridge?: WeixinJSBridge
  }
}

export {}
