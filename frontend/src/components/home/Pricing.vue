<template>
  <section id="pricing" class="scroll-mt-16 py-12 sm:py-16">
    <div>
      <h2 class="mb-3 text-center text-2xl font-bold text-gray-900 dark:text-white sm:text-3xl">
        {{ t('home.pricing.title') }}
      </h2>
      <p class="mb-8 text-center text-sm text-gray-500 dark:text-dark-400 sm:mb-12 sm:text-base">
        {{ t('home.pricing.subtitle') }}
      </p>

      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <!-- No Plans / Simple Mode -->
      <div v-else-if="!plans.length">
        <div class="mx-auto max-w-lg rounded-2xl border border-gray-200/50 bg-white/60 p-8 text-center backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/60">
          <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-900/30">
            <Icon name="check" size="lg" class="text-emerald-500" />
          </div>
          <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">{{ t('home.pricing.freeMode') }}</h3>
          <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('home.pricing.freeModeDesc') }}</p>
        </div>
        <div class="mt-6 text-center">
          <button
            @click="showContactDialog = true"
            class="text-sm text-primary-500 transition-colors hover:text-primary-600 hover:underline dark:text-primary-400"
          >
            {{ t('home.pricing.contactSales') }}
          </button>
        </div>
      </div>

      <!-- Plan Cards -->
      <div v-else>
        <div class="grid gap-6" :class="gridColsClass">
          <div
            v-for="(plan, i) in plans"
            :key="plan.id"
            class="relative flex flex-col rounded-2xl border-2 p-6 transition-all"
            :class="isHighlighted(i)
              ? 'border-primary-500 bg-white shadow-xl shadow-primary-500/10 dark:bg-dark-800'
              : 'border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800'"
          >
            <!-- Recommended badge -->
            <div v-if="isHighlighted(i)" class="absolute -top-3 right-4">
              <span class="rounded-full bg-primary-500 px-3 py-1 text-xs font-semibold text-white">{{ t('home.pricing.recommended') }}</span>
            </div>

            <h3 class="mb-3 text-lg font-bold text-gray-900 dark:text-white">{{ plan.name }}</h3>

            <!-- Price -->
            <div class="mb-4">
              <span class="text-4xl font-bold text-primary-600 dark:text-primary-400">¥{{ plan.price_cny }}</span>
              <span class="text-sm text-gray-500 dark:text-dark-400">{{ t('home.pricing.perMonth') }}</span>
            </div>

            <p v-if="plan.purchasable_description" class="mb-4 text-xs text-gray-500 dark:text-dark-400">{{ plan.purchasable_description }}</p>

            <!-- Limits -->
            <ul class="mb-6 space-y-2 text-sm text-gray-600 dark:text-dark-300">
              <li v-if="plan.daily_limit_usd" class="flex items-center gap-2">
                <Icon name="check" size="sm" class="text-emerald-500" />
                {{ t('home.pricing.dailyLimit') }} ${{ plan.daily_limit_usd }}
              </li>
              <li v-if="plan.monthly_limit_usd" class="flex items-center gap-2">
                <Icon name="check" size="sm" class="text-emerald-500" />
                {{ t('home.pricing.monthlyLimit') }} ${{ plan.monthly_limit_usd }}
              </li>
            </ul>

            <!-- Spacer to push button to bottom -->
            <div class="mt-auto">
              <router-link
                :to="isAuthenticated ? '/subscriptions' : '/register'"
                class="block w-full rounded-xl py-2.5 text-center text-sm font-semibold transition-all"
                :class="isHighlighted(i)
                  ? 'bg-primary-500 text-white hover:bg-primary-600'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-white dark:hover:bg-dark-600'"
              >
                {{ isAuthenticated ? t('home.pricing.subscribe') : t('home.pricing.getStarted') }}
              </router-link>
            </div>
          </div>
        </div>

        <!-- Contact Sales -->
        <div class="mt-8 text-center">
          <button
            @click="showContactDialog = true"
            class="text-sm text-primary-500 transition-colors hover:text-primary-600 hover:underline dark:text-primary-400"
          >
            {{ t('home.pricing.contactSales') }}
          </button>
        </div>
      </div>

      <!-- Contact Dialog -->
      <ContactQRCodeModal
        :show="showContactDialog"
        :wechat-q-r-code="contactQRCodeWechat"
        :group-q-r-code="contactQRCodeGroup"
        :contact-info="contactInfo"
        @close="showContactDialog = false"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import { subscriptionPlanAPI } from '@/api/subscriptionPlan'
import Icon from '@/components/icons/Icon.vue'
import ContactQRCodeModal from '@/components/common/ContactQRCodeModal.vue'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const isAuthenticated = computed(() => authStore.isAuthenticated)
const contactQRCodeWechat = computed(() => appStore.contactQRCodeWechat)
const contactQRCodeGroup = computed(() => appStore.contactQRCodeGroup)
const contactInfo = computed(() => appStore.contactInfo)
const loading = ref(true)
const showContactDialog = ref(false)

interface Plan {
  id: number
  name: string
  description: string | null
  purchasable_description: string | null
  price_cny: number
  validity_days: number
  daily_limit_usd: number | null
  weekly_limit_usd: number | null
  monthly_limit_usd: number | null
}

const plans = ref<Plan[]>([])

const gridColsClass = computed(() => {
  const map: Record<number, string> = {
    1: 'sm:grid-cols-1',
    2: 'sm:grid-cols-2',
    3: 'sm:grid-cols-3',
  }
  return map[plans.value.length] || 'sm:grid-cols-2 lg:grid-cols-3'
})

function isHighlighted(index: number): boolean {
  if (plans.value.length <= 1) return false
  if (plans.value.length === 2) return index === 1
  // Highlight the middle plan as recommended
  return index === Math.floor(plans.value.length / 2)
}

onMounted(async () => {
  try {
    const resp = await subscriptionPlanAPI.listPlans()
    plans.value = resp.plans || []
  } catch {
    // silently handle - show no plans state
  } finally {
    loading.value = false
  }
})
</script>
