<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ hasPassword ? t('profile.changePassword') : t('profile.setPassword') }}
      </h2>
    </div>
    <div class="px-6 py-6">
      <!-- 已有密码：修改密码表单 -->
      <form v-if="hasPassword" @submit.prevent="handleChangePassword" class="space-y-4">
        <div>
          <label for="old_password" class="input-label">
            {{ t('profile.currentPassword') }}
          </label>
          <input
            id="old_password"
            v-model="changeForm.old_password"
            type="password"
            required
            autocomplete="current-password"
            class="input"
          />
        </div>

        <div>
          <label for="new_password" class="input-label">
            {{ t('profile.newPassword') }}
          </label>
          <input
            id="new_password"
            v-model="changeForm.new_password"
            type="password"
            required
            autocomplete="new-password"
            class="input"
          />
          <p class="input-hint">
            {{ t('profile.passwordHint') }}
          </p>
        </div>

        <div>
          <label for="confirm_password" class="input-label">
            {{ t('profile.confirmNewPassword') }}
          </label>
          <input
            id="confirm_password"
            v-model="changeForm.confirm_password"
            type="password"
            required
            autocomplete="new-password"
            class="input"
          />
          <p
            v-if="changeForm.new_password && changeForm.confirm_password && changeForm.new_password !== changeForm.confirm_password"
            class="input-error-text"
          >
            {{ t('profile.passwordsNotMatch') }}
          </p>
        </div>

        <div class="flex justify-end pt-4">
          <button type="submit" :disabled="loading" class="btn btn-primary">
            {{ loading ? t('profile.changingPassword') : t('profile.changePasswordButton') }}
          </button>
        </div>
      </form>

      <!-- 未有密码：设置密码 -->
      <template v-else>
        <!-- 邮箱仍为合成邮箱，提示先绑定 -->
        <div v-if="isSyntheticEmail" class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('profile.bindEmailFirst') }}
        </div>

        <!-- 已有真实邮箱，显示设置密码表单 -->
        <div v-else>
          <p class="mb-4 text-sm text-gray-500 dark:text-gray-400">
            {{ t('profile.setPasswordDescription') }}
          </p>
          <form @submit.prevent="handleSetPassword" class="space-y-4">
            <div>
              <label for="verify_code" class="input-label">
                {{ t('profile.emailVerifyCode') }}
              </label>
              <div class="flex gap-3">
                <input
                  id="verify_code"
                  v-model="setForm.verify_code"
                  type="text"
                  inputmode="numeric"
                  required
                  autocomplete="off"
                  :placeholder="t('profile.verifyCodePlaceholder')"
                  class="input flex-1"
                />
                <button
                  type="button"
                  :disabled="sendingCode || countdown > 0"
                  class="btn btn-secondary whitespace-nowrap"
                  @click="handleSendCode"
                >
                  <template v-if="sendingCode">{{ t('profile.sendingVerifyCode') }}</template>
                  <template v-else-if="countdown > 0">{{ t('profile.verifyCodeCountdown', { seconds: countdown }) }}</template>
                  <template v-else>{{ t('profile.sendVerifyCode') }}</template>
                </button>
              </div>
            </div>

            <div>
              <label for="set_new_password" class="input-label">
                {{ t('profile.newPassword') }}
              </label>
              <input
                id="set_new_password"
                v-model="setForm.new_password"
                type="password"
                required
                autocomplete="new-password"
                class="input"
              />
              <p class="input-hint">
                {{ t('profile.passwordHint') }}
              </p>
            </div>

            <div>
              <label for="set_confirm_password" class="input-label">
                {{ t('profile.confirmNewPassword') }}
              </label>
              <input
                id="set_confirm_password"
                v-model="setForm.confirm_password"
                type="password"
                required
                autocomplete="new-password"
                class="input"
              />
              <p
                v-if="setForm.new_password && setForm.confirm_password && setForm.new_password !== setForm.confirm_password"
                class="input-error-text"
              >
                {{ t('profile.passwordsNotMatch') }}
              </p>
            </div>

            <div class="flex justify-end pt-4">
              <button type="submit" :disabled="loading" class="btn btn-primary">
                {{ loading ? t('profile.settingPassword') : t('profile.setPasswordButton') }}
              </button>
            </div>
          </form>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { userAPI } from '@/api'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const user = computed(() => authStore.user)
const hasPassword = computed(() => user.value?.has_password ?? true)
const isSyntheticEmail = computed(() => {
  const email = user.value?.email?.toLowerCase() || ''
  return email.endsWith('@wechat-auth.invalid') || email.endsWith('@linuxdo.invalid')
})

const loading = ref(false)
const sendingCode = ref(false)
const countdown = ref(0)
let countdownTimer: ReturnType<typeof setInterval> | null = null

// 修改密码表单
const changeForm = ref({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

// 设置密码表单
const setForm = ref({
  verify_code: '',
  new_password: '',
  confirm_password: ''
})

const handleChangePassword = async () => {
  if (changeForm.value.new_password !== changeForm.value.confirm_password) {
    appStore.showError(t('profile.passwordsNotMatch'))
    return
  }

  if (changeForm.value.new_password.length < 8) {
    appStore.showError(t('profile.passwordTooShort'))
    return
  }

  loading.value = true
  try {
    await userAPI.changePassword(changeForm.value.old_password, changeForm.value.new_password)
    changeForm.value = { old_password: '', new_password: '', confirm_password: '' }
    appStore.showSuccess(t('profile.passwordChangeSuccess'))
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('profile.passwordChangeFailed'))
  } finally {
    loading.value = false
  }
}

const handleSendCode = async () => {
  sendingCode.value = true
  try {
    const result = await userAPI.sendSetPasswordCode()
    appStore.showSuccess(t('profile.verifyCodeSent'))
    countdown.value = result.countdown || 60
    countdownTimer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        if (countdownTimer) clearInterval(countdownTimer)
        countdownTimer = null
      }
    }, 1000)
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || error.response?.data?.message || t('profile.setPasswordFailed'))
  } finally {
    sendingCode.value = false
  }
}

const handleSetPassword = async () => {
  if (setForm.value.new_password !== setForm.value.confirm_password) {
    appStore.showError(t('profile.passwordsNotMatch'))
    return
  }

  if (setForm.value.new_password.length < 8) {
    appStore.showError(t('profile.passwordTooShort'))
    return
  }

  if (!setForm.value.verify_code) {
    appStore.showError(t('profile.emailVerifyCode'))
    return
  }

  loading.value = true
  try {
    await userAPI.setPassword(
      user.value!.email,
      setForm.value.verify_code,
      setForm.value.new_password
    )
    setForm.value = { verify_code: '', new_password: '', confirm_password: '' }
    appStore.showSuccess(t('profile.setPasswordSuccess'))
    // 刷新用户信息以更新 has_password 状态
    await authStore.refreshUser()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('profile.setPasswordFailed'))
  } finally {
    loading.value = false
  }
}

onUnmounted(() => {
  if (countdownTimer) clearInterval(countdownTimer)
})
</script>
