<template>
  <div
    class="relative flex min-h-screen flex-col overflow-hidden bg-gradient-to-br from-gray-50 via-primary-50/30 to-gray-100 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950"
  >
    <!-- Background Decorations -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div
        class="absolute -right-40 -top-40 h-96 w-96 rounded-full bg-primary-400/20 blur-3xl"
      ></div>
      <div
        class="absolute -bottom-40 -left-40 h-96 w-96 rounded-full bg-primary-500/15 blur-3xl"
      ></div>
      <div
        class="absolute left-1/3 top-1/4 h-72 w-72 rounded-full bg-primary-300/10 blur-3xl"
      ></div>
      <div
        class="absolute bottom-1/4 right-1/4 h-64 w-64 rounded-full bg-primary-400/10 blur-3xl"
      ></div>
      <div
        class="absolute inset-0 bg-[linear-gradient(rgba(217,119,87,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(217,119,87,0.03)_1px,transparent_1px)] bg-[size:64px_64px]"
      ></div>
    </div>

    <!-- Header -->
    <header class="relative z-20 px-6 py-4">
      <nav class="mx-auto flex max-w-6xl items-center justify-between">
        <!-- Logo -->
        <router-link to="/home" class="flex items-center gap-3">
          <div class="h-10 w-10 overflow-hidden rounded-xl shadow-md">
            <img :src="currentLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="text-lg font-semibold text-gray-900 dark:text-white">{{ siteName }}</span>
        </router-link>

        <!-- Nav Actions -->
        <div class="flex items-center gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Theme Toggle -->
          <button
            @click="toggleTheme"
            class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t(`nav.${themeMode === 'light' ? 'darkMode' : themeMode === 'dark' ? 'autoMode' : 'lightMode'}`)"
          >
            <Icon v-if="themeMode === 'light'" name="sun" size="md" class="text-amber-500" />
            <Icon v-else-if="themeMode === 'dark'" name="moon" size="md" class="text-indigo-400" />
            <Icon v-else name="clock" size="md" class="text-emerald-500" />
          </button>

          <!-- Back to Home -->
          <router-link
            to="/home"
            class="inline-flex items-center rounded-full bg-gray-900 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            {{ t('common.back') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="relative z-10 flex-1 px-4 py-8 sm:px-6 sm:py-16">
      <div class="mx-auto max-w-6xl">
        <!-- Page Title -->
        <div class="mb-8 text-center sm:mb-12">
          <h1 class="mb-3 text-3xl font-bold text-gray-900 dark:text-white sm:mb-4 sm:text-4xl lg:text-5xl">
            {{ t('releaseNotes.title') }}
          </h1>
          <p class="mx-auto max-w-2xl text-base text-gray-600 dark:text-dark-300 sm:text-lg">
            {{ t('releaseNotes.subtitle') }}
          </p>
        </div>

        <!-- Release List -->
        <div class="space-y-6 sm:space-y-8">
          <div
            v-for="entry in releases"
            :key="entry.date"
            class="card overflow-hidden"
          >
            <div class="card-header flex items-center gap-3 sm:gap-4">
              <div
                class="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-primary-500 to-primary-600 shadow-lg shadow-primary-500/30 sm:h-12 sm:w-12"
              >
                <Icon name="badge" size="md" class="text-white" />
              </div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white sm:text-xl">
                {{ formatDate(entry.date) }}
              </h2>
            </div>

            <div class="card-body">
              <!-- Features -->
              <div v-if="entry.features.length > 0" class="mb-6">
                <h3 class="mb-3 flex items-center gap-2 text-base font-semibold text-gray-900 dark:text-white">
                  <Icon name="sparkles" size="sm" class="text-blue-500" />
                  {{ t('releaseNotes.sections.features') }}
                </h3>
                <ul class="space-y-2">
                  <li
                    v-for="(feature, index) in entry.features"
                    :key="index"
                    class="flex items-start gap-3 text-sm text-gray-700 dark:text-dark-300"
                  >
                    <Icon name="check" size="sm" class="mt-0.5 flex-shrink-0 text-blue-500" />
                    <span>{{ feature }}</span>
                  </li>
                </ul>
              </div>

              <!-- Improvements -->
              <div v-if="entry.improvements.length > 0" class="mb-6">
                <h3 class="mb-3 flex items-center gap-2 text-base font-semibold text-gray-900 dark:text-white">
                  <Icon name="arrowUp" size="sm" class="text-emerald-500" />
                  {{ t('releaseNotes.sections.improvements') }}
                </h3>
                <ul class="space-y-2">
                  <li
                    v-for="(improvement, index) in entry.improvements"
                    :key="index"
                    class="flex items-start gap-3 text-sm text-gray-700 dark:text-dark-300"
                  >
                    <Icon name="check" size="sm" class="mt-0.5 flex-shrink-0 text-emerald-500" />
                    <span>{{ improvement }}</span>
                  </li>
                </ul>
              </div>

              <!-- Bug Fixes -->
              <div v-if="entry.bugFixes.length > 0">
                <h3 class="mb-3 flex items-center gap-2 text-base font-semibold text-gray-900 dark:text-white">
                  <Icon name="exclamationCircle" size="sm" class="text-red-500" />
                  {{ t('releaseNotes.sections.bugFixes') }}
                </h3>
                <ul class="space-y-2">
                  <li
                    v-for="(fix, index) in entry.bugFixes"
                    :key="index"
                    class="flex items-start gap-3 text-sm text-gray-700 dark:text-dark-300"
                  >
                    <Icon name="check" size="sm" class="mt-0.5 flex-shrink-0 text-red-500" />
                    <span>{{ fix }}</span>
                  </li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-gray-200/50 px-6 py-8 dark:border-dark-800/50">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useTheme } from '@/composables/useTheme'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const { t, locale } = useI18n()
const appStore = useAppStore()
const { isDark, themeMode, toggleTheme } = useTheme()

// Site settings
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Code80')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteLogoDark = computed(() => appStore.cachedPublicSettings?.site_logo_dark || appStore.siteLogoDark || '')

const currentLogo = computed(() => {
  if (isDark.value && siteLogoDark.value) {
    return siteLogoDark.value
  }
  return siteLogo.value
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

interface ReleaseEntry {
  date: string
  features: string[]
  improvements: string[]
  bugFixes: string[]
}

const releases = computed<ReleaseEntry[]>(() => {
  if (locale.value === 'zh') {
    return [
      {
        date: '2026-02-08',
        features: [
          'Antigravity 添加 onboardUser 支持并修复 project_id 补齐逻辑',
          '余额批次系统完善——过期通知邮件、余额明细弹框、首页导航优化',
          '用户余额历史中整合微信充值订单记录',
          '强制绑定邮箱改为可配置系统设置',
          '微信用户强制绑定邮箱并设置密码'
        ],
        improvements: [
          '错误处理性能优化',
          '错误透传规则支持 skip_monitoring 跳过运维监控记录'
        ],
        bugFixes: [
          '修复微信登录绑定状态、邮箱绑定密码校验及余额历史分页问题',
          '修复错误透传规则 skip_monitoring 未生效的问题',
          '修复支付回调和兑换码事务污染及订单状态补偿问题'
        ]
      },
      {
        date: '2026-02-03',
        features: [
          '余额批次过期功能（Balance Lots）',
          '完善余额批次过期功能——设置 API、网关扣费和迁移重试',
          '添加账号到期提醒邮件配置 UI',
          'Claude OAuth 账号到期提醒定时任务',
          'Antigravity 转发与测试支持 daily/prod 单 URL 切换'
        ],
        improvements: [
          'MODEL_CAPACITY_EXHAUSTED 使用固定 1s 间隔重试 60 次，不切换账号',
          '同账号瞬态错误先重试再故障转移',
          '空流式响应时自动故障转移并临时取消调度',
          'Google "Invalid project resource name" 400 错误自动故障转移'
        ],
        bugFixes: [
          '移除特定 system 以适配新版 CC 客户端缓存失效的 bug',
          '移除 Antigravity apikey 账户额外的表单'
        ]
      },
      {
        date: '2026-01-27',
        features: [
          '首页导航栏添加文档链接按钮',
          '首页重构',
          '重构安装教程页为分平台分工具详细指南',
          '添加 Antigravity 单账号 503 退避重试机制'
        ],
        improvements: [
          '将 scope 级别限流重构为 model 级别限流',
          '同排序分组内随机打散账号，防止惊群效应',
          'Antigravity 账号故障转移之间添加线性延迟',
          '统一 Antigravity + Gemini 错误策略并支持自定义错误码'
        ],
        bugFixes: [
          '单账号分组首次 503 不设模型限流标记，避免后续请求雪崩',
          '修复粘性会话故障转移时缓存计费豁免问题',
          '修复 Gemini 原生请求格式解析导致的会话哈希生成错误',
          '防止不同用户相同消息的 sessionHash 碰撞'
        ]
      },
      {
        date: '2026-01-21',
        features: [
          '普通用户新手引导流程',
          '管理后台分组拖拽排序',
          '用户列表页显示当前并发数',
          'OpenAI OAuth 账号支持批量 RT 输入创建',
          '首页 SaaS 落地页重设计 + 安装教程页重构'
        ],
        improvements: [
          '将 upstream 账号类型重构为 apikey，自动追加 /antigravity',
          '客户端透传所有请求头，取代手动 header 设置',
          '检测流式传输中客户端断开连接并继续消耗上游数据用于计费',
          '将基于 Trie 的 digest 会话存储替换为扁平缓存'
        ],
        bugFixes: [
          '修复创建账户表单默认优先级从 1 改为 50，与后端 schema 一致',
          '修复前端 upstream 账户编辑字段和创建时 mixed_scheduling 问题',
          '修复 upstream 账户转发的专用方法'
        ]
      },
      {
        date: '2026-01-17',
        features: [
          '智能重试最多 1 次 + 失败时清除粘性会话',
          '添加 Anthropic 粘性会话 digest chain 匹配',
          'Antigravity 全面增强——模型映射、限流、调度和运维',
          '全局错误透传规则功能',
          '订阅套餐升级功能（从低价套餐升级到高价套餐）'
        ],
        improvements: [
          '简化粘性会话限流处理——任何限流立即切换',
          '移除 Anthropic digest chain 从 Messages handler',
          '前端限流时间显示精确到秒'
        ],
        bugFixes: [
          '修复管理页面活跃会话数始终显示为 0 的问题',
          '修复 codex 更新用量窗口异常的 bug',
          '修复 Antigravity 429 回退冷却时间从 5 分钟改为 30 秒',
          '修复 Antigravity 自动修复 max_tokens <= budget_tokens 导致的 400 错误'
        ]
      },
      {
        date: '2026-01-14',
        features: [
          '支持用户专属分组倍率配置',
          '实现 Refresh Token 机制',
          '微信绑定支持订阅号扫码模式，每日报告默认启用',
          '支持 HTTP/2 Cleartext (h2c) 连接配置',
          'API Key 独立配额和过期时间支持'
        ],
        improvements: [
          '/v1/usage 端点按 API Key 过滤统计而非 UserID',
          '增强 Gemini API 授权错误处理，自动提取并显示激活 URL',
          '代理探测添加多 URL 回退机制'
        ],
        bugFixes: [
          '修复模型前缀映射逻辑错误',
          '优化 Gemini 接口认证兼容性，支持 Authorization: Bearer',
          '移除上游请求中不支持的 safety_identifier 和 previous_response_id 字段',
          '修复 thinking 块被意外修改导致的 400 错误'
        ]
      },
      {
        date: '2026-01-11',
        features: [
          '增加邀请码注册功能',
          '微信账号类型配置支持订阅号/未认证服务号',
          '用户微信账号解绑功能',
          '管理后台用户余额/并发历史弹框',
          '从其他分组复制账号功能',
          'Gemini 200K 长上下文双倍计费功能'
        ],
        improvements: [
          '支持过滤无效 API Key 错误，不写入错误日志',
          '将 USER_INACTIVE 错误排除在 SLA 统计之外',
          '增强 /v1/usage 端点返回完整用量统计',
          '数据导入导出功能（账号、代理等）'
        ],
        bugFixes: [
          '修复 Gemini 已注册用户 OAuth 授权时错误调用 onboardUser 的问题',
          '修复 OAuth token 刷新后调度器缓存不一致问题',
          '修复 Redis TLS 配置选项位置错误',
          '修复 Gemini 工具调用缺少 thoughtSignature 导致 INVALID_ARGUMENT 错误'
        ]
      }
    ]
  } else {
    return [
      {
        date: '2026-02-08',
        features: [
          'Antigravity onboardUser support with project_id auto-fill',
          'Balance lots system improvements — expiry notifications, balance detail modal, homepage nav',
          'Integrated WeChat recharge orders into user balance history',
          'Made mandatory email binding a configurable system setting',
          'WeChat users mandatory email binding and password setup'
        ],
        improvements: [
          'Error handling performance optimization',
          'Error passthrough rules support skip_monitoring to skip ops logging'
        ],
        bugFixes: [
          'Fixed WeChat login binding status, email binding password validation and balance history pagination',
          'Fixed error passthrough rule skip_monitoring not taking effect',
          'Fixed payment callback and redeem code transaction contamination and order status compensation'
        ]
      },
      {
        date: '2026-02-03',
        features: [
          'Balance Lots expiration feature',
          'Balance lots expiration improvements — settings API, gateway billing deduction and migration retry',
          'Account expiry reminder email configuration UI',
          'Claude OAuth account expiry reminder scheduled task',
          'Antigravity forwarding and testing support for daily/prod single URL switching'
        ],
        improvements: [
          'MODEL_CAPACITY_EXHAUSTED retry with fixed 1s interval 60 times without switching accounts',
          'Same-account retry before failover for transient errors',
          'Auto failover and temp-unschedule on empty stream response',
          'Auto failover on Google "Invalid project resource name" 400 error'
        ],
        bugFixes: [
          'Removed specific system prompt to fix new CC client cache invalidation bug',
          'Removed extra form for Antigravity API key accounts'
        ]
      },
      {
        date: '2026-01-27',
        features: [
          'Added documentation link button to homepage navigation',
          'CRS sync preview and account selection',
          'Homepage redesign',
          'Rebuilt installation tutorial as platform-specific detailed guide',
          'Antigravity single-account 503 backoff retry mechanism'
        ],
        improvements: [
          'Replaced scope-level rate limiting with model-level rate limiting',
          'Shuffle accounts within same sort group to prevent thundering herd',
          'Linear delay between Antigravity account failover switches',
          'Unified error policy for Antigravity + custom error codes for Gemini accounts'
        ],
        bugFixes: [
          'First 503 in single-account group no longer sets model rate limit mark to prevent cascading failures',
          'Fixed sticky session failover cache billing exemption',
          'Fixed Gemini native request format parsing for correct session hash generation',
          'Prevented sessionHash collision for different users with same messages'
        ]
      },
      {
        date: '2026-01-21',
        features: [
          'User onboarding guide flow',
          'Admin drag-and-drop group sort order',
          'User list page shows current concurrency count',
          'OpenAI OAuth accounts support batch RT input creation',
          'Homepage SaaS landing page redesign + installation tutorial rebuild'
        ],
        improvements: [
          'Replaced upstream account type with apikey, auto-append /antigravity',
          'Passthrough all client headers instead of manual header setting',
          'Detect client disconnect during streaming and continue draining upstream for billing',
          'Replaced Trie-based digest session store with flat cache'
        ],
        bugFixes: [
          'Fixed create account form default priority from 1 to 50 to match backend schema',
          'Fixed frontend upstream account edit fields and mixed_scheduling on create',
          'Restored upstream account forwarding with dedicated methods'
        ]
      },
      {
        date: '2026-01-17',
        features: [
          'Smart retry max 1 attempt + clear sticky session on failure',
          'Anthropic sticky session digest chain matching via Trie',
          'Antigravity comprehensive enhancements — model mapping, rate limiting, scheduling & ops',
          'Global error passthrough rules',
          'Subscription plan upgrade (from lower to higher price plan)'
        ],
        improvements: [
          'Simplified sticky session rate limit handling — switch immediately on any rate limit',
          'Removed Anthropic digest chain from Messages handler',
          'Frontend rate limit time display shows seconds'
        ],
        bugFixes: [
          'Fixed admin page active sessions always showing 0',
          'Fixed codex usage window update anomaly',
          'Reduced Antigravity 429 fallback cooldown from 5min to 30s',
          'Fixed Antigravity auto-fix max_tokens <= budget_tokens causing 400 error'
        ]
      },
      {
        date: '2026-01-14',
        features: [
          'User-specific group rate multiplier configuration',
          'Refresh Token mechanism implementation',
          'WeChat binding supports subscription account QR scan mode, daily report enabled by default',
          'HTTP/2 Cleartext (h2c) connection configuration support',
          'API Key independent quota and expiration support'
        ],
        improvements: [
          '/v1/usage endpoint filters by API Key instead of UserID',
          'Enhanced Gemini API authorization error handling with automatic activation URL extraction',
          'Proxy detection with multi-URL fallback mechanism'
        ],
        bugFixes: [
          'Fixed model prefix mapping logic error',
          'Optimized Gemini API auth compatibility with Authorization: Bearer support',
          'Removed unsupported safety_identifier and previous_response_id fields from upstream requests',
          'Fixed thinking block accidental modification causing 400 error'
        ]
      },
      {
        date: '2026-01-11',
        features: [
          'Invitation code registration',
          'WeChat account type configuration for subscription/unverified official accounts',
          'User WeChat account unbind functionality',
          'Admin user balance/concurrency history modal',
          'Copy accounts from other groups',
          'Gemini 200K long context double billing'
        ],
        improvements: [
          'Filter invalid API Key errors from error logs',
          'Exclude USER_INACTIVE errors from SLA statistics',
          'Enhanced /v1/usage endpoint with full usage statistics',
          'Data import/export bundle (accounts, proxies, etc.)'
        ],
        bugFixes: [
          'Fixed registered user OAuth triggering onboardUser incorrectly for Gemini',
          'Fixed scheduler cache inconsistency after OAuth token refresh',
          'Fixed Redis TLS configuration option positioning',
          'Fixed missing thoughtSignature for Gemini tool calls causing INVALID_ARGUMENT error'
        ]
      }
    ]
  }
})

// Format date based on locale
function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  if (locale.value === 'zh') {
    return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })
  }
  return date.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
}
</script>
