<template>
  <div class="settings-page">
    <h2 class="page-title">{{ t('settings.title') }}</h2>

    <n-tabs type="line" animated>
      <n-tab-pane name="profile" :tab="t('settings.profile')">
        <n-card :title="t('settings.profileInfo')">
          <n-form :model="profileForm" :rules="profileRules" ref="profileFormRef" label-placement="top">
            <n-form-item :label="t('users.username')" path="username">
              <n-input v-model:value="profileForm.username" />
            </n-form-item>
            <n-form-item :label="t('users.role')">
              <n-input :value="authStore.user?.role" disabled />
            </n-form-item>
          </n-form>
          <template #footer>
            <n-button type="primary" @click="updateProfile" :loading="savingProfile">
              {{ t('settings.save') }}
            </n-button>
          </template>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="password" :tab="t('settings.changePassword')">
        <n-card :title="t('settings.changePassword')">
          <n-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-placement="top">
            <n-form-item :label="t('users.currentPassword')" path="old_password">
              <n-input
                v-model:value="passwordForm.old_password"
                type="password"
                show-password-on="mousedown"
                :placeholder="t('users.currentPassword')"
              />
            </n-form-item>
            <n-form-item :label="t('users.newPassword')" path="new_password">
              <n-input
                v-model:value="passwordForm.new_password"
                type="password"
                show-password-on="mousedown"
                :placeholder="t('users.newPassword')"
              />
            </n-form-item>
            <n-form-item :label="t('settings.confirmPassword')" path="confirm_password">
              <n-input
                v-model:value="passwordForm.confirm_password"
                type="password"
                show-password-on="mousedown"
                :placeholder="t('settings.confirmPassword')"
              />
            </n-form-item>
          </n-form>
          <template #footer>
            <n-button type="primary" @click="changePassword" :loading="savingPassword">
              {{ t('settings.changePassword') }}
            </n-button>
          </template>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="appearance" :tab="t('settings.appearance')">
        <n-card :title="t('settings.appearance')">
          <n-form :model="appearanceForm" label-placement="top">
            <n-form-item :label="t('settings.language')">
              <n-select
                v-model:value="appearanceForm.language"
                :options="languageOptions"
                @update:value="changeLanguage"
              />
            </n-form-item>
          </n-form>
        </n-card>
      </n-tab-pane>

      <n-tab-pane v-if="authStore.isAdmin" name="system" :tab="t('settings.system')">
        <n-card :title="t('settings.systemInfo')">
          <n-descriptions :column="2" bordered size="small">
            <n-descriptions-item :label="t('settings.version')">v1.0.0</n-descriptions-item>
            <n-descriptions-item :label="t('settings.database')">Postgres</n-descriptions-item>
            <n-descriptions-item :label="t('settings.goVersion')">Go 1.21+</n-descriptions-item>
            <n-descriptions-item :label="t('settings.frameworks')">Gin + Vue3</n-descriptions-item>
          </n-descriptions>
        </n-card>

        <n-card :title="t('settings.security')" style="margin-top: 16px">
          <n-space vertical>
            <n-text>{{ t('settings.sessionTimeout') }}: 24h</n-text>
            <n-text>{{ t('settings.csrfProtection') }}: {{ t('common.enabled') }}</n-text>
            <n-text>{{ t('settings.tlsVersion') }}: TLS 1.3</n-text>
          </n-space>
        </n-card>
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NTabs, NTabPane, NCard, NForm, NFormItem, NInput, NSelect,
  NButton, NSpace, NText, NDescriptions, NDescriptionsItem, useMessage
} from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import api from '@/api'

const { t, locale } = useI18n()
const message = useMessage()
const authStore = useAuthStore()

const profileFormRef = ref()
const passwordFormRef = ref()

const savingProfile = ref(false)
const savingPassword = ref(false)

const profileForm = ref({
  username: authStore.user?.username || ''
})

const passwordForm = ref({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const appearanceForm = ref({
  language: locale.value
})

const profileRules = {
  username: { required: true, message: t('users.usernameRequired'), trigger: 'blur' }
}

const passwordRules = computed(() => ({
  old_password: { required: true, message: t('users.currentPassword') + ' is required', trigger: 'blur' },
  new_password: { required: true, message: t('users.newPassword') + ' is required', trigger: 'blur', minLength: 6 },
  confirm_password: {
    required: true,
    message: t('settings.confirmPassword') + ' is required',
    trigger: 'blur',
    validator: (_rule: any, value: string) => {
      if (value !== passwordForm.value.new_password) {
        return new Error(t('settings.passwordMismatch'))
      }
      return true
    }
  }
}))

const languageOptions = [
  { label: '简体中文', value: 'zh-CN' },
  { label: 'English', value: 'en-US' }
]

async function updateProfile() {
  try {
    await profileFormRef.value?.validate()
  } catch {
    return
  }

  savingProfile.value = true
  try {
    await api.put(`/users/${authStore.user?.id}`, { username: profileForm.value.username })
    authStore.user && (authStore.user.username = profileForm.value.username)
    message.success(t('settings.profileUpdated'))
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('settings.updateFailed'))
  } finally {
    savingProfile.value = false
  }
}

async function changePassword() {
  try {
    await passwordFormRef.value?.validate()
  } catch {
    return
  }

  savingPassword.value = true
  try {
    await api.post(`/users/${authStore.user?.id}/password`, {
      old_password: passwordForm.value.old_password,
      new_password: passwordForm.value.new_password
    })
    message.success(t('settings.passwordChanged'))
    passwordForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('settings.passwordChangeFailed'))
  } finally {
    savingPassword.value = false
  }
}

function changeLanguage(lang: string) {
  locale.value = lang
  localStorage.setItem('locale', lang)
}
</script>

<style scoped>
.settings-page {
  max-width: 800px;
}

.page-title {
  margin: 0 0 24px 0;
  font-size: 24px;
  font-weight: 600;
}
</style>