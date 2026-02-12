import type { SystemSettings } from '@/api/admin/settings'

export type SettingsForm = SystemSettings & {
  smtp_password: string
  turnstile_secret_key: string
  linuxdo_connect_client_secret: string
  wechat_server_token: string
  wechat_app_secret: string
}
