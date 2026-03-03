<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8">
      <!-- Logo -->
      <div>
        <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
          {{ $t('auth.ldap.title') }}
        </h2>
        <p class="mt-2 text-center text-sm text-gray-600">
          {{ $t('auth.ldap.subtitle') }}
        </p>
      </div>

      <!-- Login Form -->
      <form class="mt-8 space-y-6" @submit.prevent="handleSubmit">
        <!-- Turnstile Widget -->
        <TurnstileWidget
          v-if="turnstileEnabled"
          ref="turnstileRef"
          :site-key="turnstileSiteKey"
          @verify="onTurnstileVerify"
          @expire="onTurnstileExpire"
          @error="onTurnstileError"
        />

        <!-- Turnstile 加载错误提示 -->
        <div v-if="turnstileLoadError" class="rounded-md bg-yellow-50 p-4">
          <div class="flex">
            <div class="ml-3">
              <h3 class="text-sm font-medium text-yellow-800">
                {{ $t('auth.turnstileLoadError') }}
              </h3>
              <div class="mt-2">
                <button
                  type="button"
                  @click="reloadTurnstile"
                  class="text-sm font-medium text-yellow-800 hover:text-yellow-700 underline"
                >
                  {{ $t('common.refresh') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="rounded-md shadow-sm -space-y-px">
          <!-- Username Input -->
          <div>
            <label for="username" class="sr-only">{{ $t('auth.ldap.username') }}</label>
            <input
              id="username"
              v-model="form.username"
              name="username"
              type="text"
              required
              class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-t-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
              :placeholder="$t('auth.ldap.usernamePlaceholder')"
            />
          </div>

          <!-- Password Input -->
          <div>
            <label for="password" class="sr-only">{{ $t('auth.ldap.password') }}</label>
            <input
              id="password"
              v-model="form.password"
              name="password"
              type="password"
              required
              class="appearance-none rounded-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-b-md focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
              :placeholder="$t('auth.ldap.passwordPlaceholder')"
            />
          </div>
        </div>

        <!-- 错误提示 -->
        <div v-if="error" class="rounded-md bg-red-50 p-4">
          <div class="flex">
            <div class="ml-3">
              <h3 class="text-sm font-medium text-red-800">
                {{ error }}
              </h3>
            </div>
          </div>
        </div>

        <!-- Submit Button -->
        <div>
          <button
            type="submit"
            :disabled="loading || (turnstileEnabled && !turnstileToken)"
            class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
          >
            {{ loading ? $t('auth.ldap.signingIn') : $t('auth.ldap.signIn') }}
          </button>
        </div>

        <!-- Back Link -->
        <div class="text-center">
          <router-link
            to="/login"
            class="font-medium text-indigo-600 hover:text-indigo-500"
          >
            {{ $t('auth.ldap.backToEmailLogin') }}
          </router-link>
        </div>
      </form>
    </div>

    <!-- 2FA Modal -->
    <TotpLoginModal
      v-if="showTotpModal"
      v-model:show="showTotpModal"
      :temp-token="tempToken"
      :user-email-masked="userEmailMasked"
      @verify="handle2FAVerify"
      @cancel="handle2FACancel"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import TotpLoginModal from '@/components/auth/TotpLoginModal.vue'
import { getPublicSettings, isTotp2FARequired } from '@/api/auth'
import { useLdapAuth } from '../store'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { loginWithLdap } = useLdapAuth()
const { t } = useI18n()

const form = ref({
  username: '',
  password: ''
})

const loading = ref(false)
const error = ref('')

// Turnstile 相关状态
const turnstileEnabled = ref(false)
const turnstileSiteKey = ref('')
const turnstileToken = ref('')
const turnstileRef = ref<InstanceType<typeof TurnstileWidget>>()
const turnstileLoadError = ref(false)

// 2FA 相关状态
const showTotpModal = ref(false)
const tempToken = ref('')
const userEmailMasked = ref('')

// 加载公共配置
onMounted(async () => {
  try {
    const settings = await getPublicSettings()
    turnstileEnabled.value = settings.turnstile_enabled
    turnstileSiteKey.value = settings.turnstile_site_key || ''
  } catch (err) {
    console.error('Failed to load public settings:', err)
  }
})

// Turnstile 事件处理
function onTurnstileVerify(token: string) {
  turnstileToken.value = token
  turnstileLoadError.value = false
}

function onTurnstileExpire() {
  turnstileToken.value = ''
}

function onTurnstileError() {
  console.error('Turnstile error: Failed to load or verify')
  turnstileLoadError.value = true
  turnstileToken.value = ''
}

function reloadTurnstile() {
  turnstileLoadError.value = false
  turnstileRef.value?.reset()
}

async function handleSubmit() {
  error.value = ''
  loading.value = true

  try {
    const response = await loginWithLdap(
      form.value.username,
      form.value.password,
      turnstileToken.value
    )

    // 检查是否需要 2FA
    if (isTotp2FARequired(response)) {
      tempToken.value = response.temp_token!
      userEmailMasked.value = response.user_email_masked!
      showTotpModal.value = true
      return
    }

    // 直接登录成功，跳转
    router.push((route.query.redirect as string) || '/')
  } catch (err: any) {
    error.value = err.response?.data?.message || t('auth.ldap.loginFailed')
    turnstileRef.value?.reset()
  } finally {
    loading.value = false
  }
}

async function handle2FAVerify(totpCode: string) {
  try {
    loading.value = true
    await authStore.login2FA(tempToken.value, totpCode)
    showTotpModal.value = false
    router.push((route.query.redirect as string) || '/')
  } catch (err: any) {
    error.value = err.response?.data?.message || t('auth.ldap.loginFailed')
  } finally {
    loading.value = false
  }
}

function handle2FACancel() {
  showTotpModal.value = false
  tempToken.value = ''
  userEmailMasked.value = ''
}
</script>
