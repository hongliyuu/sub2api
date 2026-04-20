<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t("profile.editProfile") }}
      </h2>
    </div>
    <div class="px-6 py-6">
      <form @submit.prevent="handleUpdateProfile" class="space-y-4">
        <div>
          <label class="input-label">
            {{ t("profile.avatar") }}
          </label>
          <ImageUpload
            :model-value="avatarDataUrl"
            :upload-label="t('profile.uploadAvatar')"
            :remove-label="t('profile.removeAvatar')"
            :hint="t('profile.avatarHint')"
            :max-size="100 * 1024"
            @update:model-value="handleAvatarChange"
          />
        </div>

        <div>
          <label for="username" class="input-label">
            {{ t("profile.username") }}
          </label>
          <input
            id="username"
            v-model="username"
            type="text"
            class="input"
            :placeholder="t('profile.enterUsername')"
          />
        </div>

        <div class="flex justify-end pt-4">
          <button type="submit" :disabled="loading" class="btn btn-primary">
            {{ loading ? t("profile.updating") : t("profile.updateProfile") }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useAuthStore } from "@/stores/auth";
import { useAppStore } from "@/stores/app";
import { userAPI } from "@/api";
import ImageUpload from "@/components/common/ImageUpload.vue";

const props = defineProps<{
  initialUsername: string;
  initialAvatarUrl: string;
}>();

const { t } = useI18n();
const authStore = useAuthStore();
const appStore = useAppStore();

const username = ref(props.initialUsername);
const avatarDataUrl = ref(props.initialAvatarUrl);
const avatarDirty = ref(false);
const loading = ref(false);

watch(
  () => props.initialUsername,
  (val) => {
    username.value = val;
  },
);

watch(
  () => props.initialAvatarUrl,
  (val) => {
    if (!avatarDirty.value) {
      avatarDataUrl.value = val;
    }
  },
);

const handleAvatarChange = (value: string) => {
  avatarDataUrl.value = value;
  avatarDirty.value = true;
};

const handleUpdateProfile = async () => {
  if (!username.value.trim()) {
    appStore.showError(t("profile.usernameRequired"));
    return;
  }

  loading.value = true;
  try {
    const updatedUser = await userAPI.updateProfile({
      username: username.value,
      ...(avatarDirty.value ? { avatar_data_url: avatarDataUrl.value } : {}),
    });
    authStore.user = updatedUser;
    avatarDataUrl.value = updatedUser.avatar_url || "";
    avatarDirty.value = false;
    appStore.showSuccess(t("profile.updateSuccess"));
  } catch (error: any) {
    appStore.showError(
      error.response?.data?.detail || t("profile.updateFailed"),
    );
  } finally {
    loading.value = false;
  }
};
</script>
