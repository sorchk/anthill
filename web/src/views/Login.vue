<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <n-icon size="48" depth="3">
          <server-outline />
        </n-icon>
        <h1>{{ t('login.title') }}</h1>
        <p>{{ t('login.subtitle') }}</p>
      </div>

      <n-form
        ref="formRef"
        :model="formValue"
        :rules="rules"
        size="large"
        @submit.prevent="handleLogin"
      >
        <n-form-item path="username" :show-label="false">
          <n-input
            v-model:value="formValue.username"
            :placeholder="t('login.username')"
            :maxlength="32"
            clearable
          >
            <template #prefix>
              <n-icon><person-outline /></n-icon>
            </template>
          </n-input>
        </n-form-item>

        <n-form-item path="password" :show-label="false">
          <n-input
            v-model:value="formValue.password"
            type="password"
            :placeholder="t('login.password')"
            show-password-on="mousedown"
            :maxlength="64"
            @keydown.enter="handleLogin"
          >
            <template #prefix>
              <n-icon><lock-closed-outline /></n-icon>
            </template>
          </n-input>
        </n-form-item>

        <n-button
          type="primary"
          attr-type="submit"
          block
          :loading="loading"
          :disabled="loading"
          :render-icon="renderIcon"
        >
          {{ loading ? t('login.loggingIn') : t('login.signIn') }}
        </n-button>
      </n-form>

      <div class="login-footer">
        <n-text depth="3">{{ t('login.defaultCredentials') }}</n-text>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, h } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NForm, NFormItem, NInput, NButton, NIcon,
  NText, useMessage, useNotification
} from 'naive-ui'
import { ServerOutline, PersonOutline, LockClosedOutline, ArrowForwardOutline } from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const { t } = useI18n()
const message = useMessage()
const notification = useNotification()
const authStore = useAuthStore()

const formRef = ref()
const loading = ref(false)

const formValue = ref({
  username: '',
  password: ''
})

const rules = {
  username: { required: true, message: t('login.username'), trigger: 'blur' },
  password: { required: true, message: t('login.password'), trigger: 'blur' }
}

function renderIcon() {
  return h(NIcon, null, { default: () => h(ArrowForwardOutline) })
}

async function handleLogin() {
  if (loading.value) return

  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  loading.value = true

  try {
    await authStore.login(formValue.value.username, formValue.value.password)

    notification.success({
      content: t('login.welcome'),
      meta: t('login.loggedInAs', { username: formValue.value.username }),
      duration: 2000
    })

    router.push('/')
  } catch (error: any) {
    message.error(error?.response?.data?.error || t('login.loginFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
}

.login-card {
  width: 100%;
  max-width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-header .n-icon {
  color: #18a058;
  margin-bottom: 16px;
}

.login-header h1 {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  margin: 0 0 8px;
}

.login-header p {
  font-size: 14px;
  color: #999;
  margin: 0;
}

.login-footer {
  text-align: center;
  margin-top: 24px;
}
</style>