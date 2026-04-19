<template>
  <AppLayout>
    <div class="mx-auto max-w-4xl space-y-6">
      <div class="grid grid-cols-1 gap-6 sm:grid-cols-3">
        <StatCard :title="t('profile.accountBalance')" :value="formatCurrency(user?.balance || 0)" :icon="WalletIcon" icon-variant="success" />
        <StatCard :title="t('profile.concurrencyLimit')" :value="user?.concurrency || 0" :icon="BoltIcon" icon-variant="warning" />
        <StatCard :title="t('profile.memberSince')" :value="formatDate(user?.created_at || '', { year: 'numeric', month: 'long' })" :icon="CalendarIcon" icon-variant="primary" />
      </div>
      <ProfileInfoCard :user="user" />
      <div v-if="contactInfo" class="card border-primary-200 bg-primary-50 dark:bg-primary-900/20 p-6">
        <div class="flex items-center gap-4">
          <div class="p-3 bg-primary-100 rounded-xl text-primary-600"><Icon name="chat" size="lg" /></div>
          <div><h3 class="font-semibold text-primary-800 dark:text-primary-200">{{ t('common.contactSupport') }}</h3><p class="text-sm font-medium">{{ contactInfo }}</p></div>
        </div>
      </div>
      <ProfileEditForm :initial-username="user?.username || ''" />
      <ProfileBalanceNotifyCard
        v-if="user && balanceLowNotifyEnabled"
        :enabled="user.balance_notify_enabled ?? true"
        :threshold="user.balance_notify_threshold"
        :extra-emails="user.balance_notify_extra_emails ?? []"
        :system-default-threshold="systemDefaultThreshold"
        :user-email="user.email"
      />
      <!-- Referral Card -->
      <div v-if="referralEnabled" class="card p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-2 bg-primary-100 dark:bg-primary-900/30 rounded-lg text-primary-600 dark:text-primary-400">
            <Icon name="gift" size="lg" />
          </div>
          <div>
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ t('profile.referralTitle') }}</h3>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('profile.referralDesc') }}</p>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-4 mb-4">
          <div class="rounded-xl bg-gray-50 dark:bg-dark-800 p-3 text-center">
            <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ user?.aff_count ?? 0 }}</div>
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('profile.referralCount') }}</div>
          </div>
          <div class="rounded-xl bg-gray-50 dark:bg-dark-800 p-3 text-center">
            <div class="text-2xl font-bold text-gray-900 dark:text-white">${{ (user?.aff_history_balance ?? 0).toFixed(2) }}</div>
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('profile.referralEarned') }}</div>
          </div>
        </div>
        <div v-if="affCode" class="space-y-2">
          <label class="text-sm font-medium text-gray-700 dark:text-dark-300">{{ t('profile.referralLink') }}</label>
          <div class="flex gap-2">
            <input
              readonly
              :value="referralLink"
              class="input flex-1 text-sm font-mono"
            />
            <button
              @click="copyReferralLink"
              class="btn btn-secondary whitespace-nowrap"
            >
              {{ copied ? t('common.copied') : t('common.copy') }}
            </button>
          </div>
        </div>
        <div v-else class="text-sm text-gray-400 dark:text-dark-500">{{ t('profile.referralLinkLoading') }}</div>
      </div>
      <ProfilePasswordForm />
      <ProfileTotpCard />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, h, onMounted } from 'vue'; import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'; import { formatDate } from '@/utils/format'
import { authAPI, userAPI } from '@/api'; import AppLayout from '@/components/layout/AppLayout.vue'
import StatCard from '@/components/common/StatCard.vue'
import ProfileInfoCard from '@/components/user/profile/ProfileInfoCard.vue'
import ProfileEditForm from '@/components/user/profile/ProfileEditForm.vue'
import ProfileBalanceNotifyCard from '@/components/user/profile/ProfileBalanceNotifyCard.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import { Icon } from '@/components/icons'

const { t } = useI18n(); const authStore = useAuthStore(); const user = computed(() => authStore.user)
const contactInfo = ref('')
const balanceLowNotifyEnabled = ref(false)
const systemDefaultThreshold = ref(0)
const referralEnabled = ref(false)
const affCode = ref('')
const copied = ref(false)

const referralLink = computed(() => {
  if (!affCode.value) return ''
  const base = window.location.origin
  return `${base}/register?aff=${affCode.value}`
})

async function copyReferralLink() {
  if (!referralLink.value) return
  try {
    await navigator.clipboard.writeText(referralLink.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    // ignore
  }
}

const WalletIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'M21 12a2.25 2.25 0 00-2.25-2.25H15a3 3 0 11-6 0H5.25A2.25 2.25 0 003 12' })]) }
const BoltIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'm3.75 13.5 10.5-11.25L12 10.5h8.25L9.75 21.75 12 13.5H3.75z' })]) }
const CalendarIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'M6.75 3v2.25M17.25 3v2.25' })]) }

onMounted(async () => {
  try {
    const s = await authAPI.getPublicSettings()
    contactInfo.value = s.contact_info || ''
    balanceLowNotifyEnabled.value = s.balance_low_notify_enabled ?? false
    systemDefaultThreshold.value = s.balance_low_notify_threshold ?? 0
    referralEnabled.value = s.referral_enabled ?? false
    if (referralEnabled.value) {
      try {
        const res = await userAPI.getAffCode()
        affCode.value = res.aff_code || ''
      } catch {
        // ignore
      }
    }
  } catch (error) { console.error('Failed to load settings:', error) }
})
const formatCurrency = (v: number) => `$${v.toFixed(2)}`
</script>