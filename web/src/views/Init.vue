<template>
  <div class="init-page">
    <div class="init-card">
      <div class="init-header">
        <n-icon size="48" depth="3">
          <server-outline />
        </n-icon>
        <h1>{{ t('init.welcomeTitle') }}</h1>
        <p>{{ t('init.welcomeSubtitle') }}</p>
      </div>

      <n-form
        v-if="step === 1"
        ref="formRef"
        :model="formValue"
        :rules="rules"
        size="large"
        @submit.prevent="handleNext"
      >
        <n-form-item path="username" :label="t('init.username')">
          <n-input
            v-model:value="formValue.username"
            :placeholder="t('init.username')"
            :maxlength="32"
            clearable
          >
            <template #prefix>
              <n-icon><person-outline /></n-icon>
            </template>
          </n-input>
        </n-form-item>

        <n-form-item path="password" :label="t('init.password')">
          <n-input
            v-model:value="formValue.password"
            type="password"
            :placeholder="t('init.password')"
            show-password-on="mousedown"
            :maxlength="64"
          >
            <template #prefix>
              <n-icon><lock-closed-outline /></n-icon>
            </template>
          </n-input>
        </n-form-item>

        <n-form-item path="confirmPassword" :label="t('init.confirmPassword')">
          <n-input
            v-model:value="formValue.confirmPassword"
            type="password"
            :placeholder="t('init.confirmPassword')"
            show-password-on="mousedown"
            :maxlength="64"
            @keydown.enter="handleNext"
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
          @click="handleNext"
        >
          {{ loading ? t('init.creating') : t('init.next') }}
        </n-button>
      </n-form>

      <div v-else-if="step === 2" class="init-success">
        <n-icon size="64" color="#18a058">
          <checkmark-circle-outline />
        </n-icon>
        <p>{{ t('init.createSuccess') }}</p>
        <p class="redirect-text">{{ t('init.redirecting') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NForm, NFormItem, NInput, NButton, NIcon,
  useMessage
} from 'naive-ui'
import { ServerOutline, PersonOutline, LockClosedOutline, CheckmarkCircleOutline } from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const { t } = useI18n()
const message = useMessage()
const authStore = useAuthStore()

const step = ref(1)
const loading = ref(false)
const formRef = ref()

const formValue = ref({
  username: '',
  password: '',
  confirmPassword: ''
})

const rules = {
  username: { required: true, message: t('init.usernameRequired'), trigger: 'blur' },
  password: [
    { required: true, message: t('init.passwordRequired'), trigger: 'blur' },
    { min: 6, message: t('init.passwordMinLength'), trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: t('init.passwordRequired'), trigger: 'blur' },
    {
      validator: (_rule: any, value: string) => value === formValue.value.password,
      message: t('init.passwordMismatch'),
      trigger: 'blur'
    }
  ]
}

async function handleNext() {
  if (loading.value) return

  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  loading.value = true

  try {
    await authStore.initAdmin(
      formValue.value.username,
      formValue.value.password,
      formValue.value.confirmPassword
    )
    step.value = 2
    setTimeout(() => {
      router.push('/login')
    }, 3000)
  } catch (error: any) {
    const errorMsg = error?.response?.data?.error || t('init.createFailed')
    if (errorMsg.includes('already exists')) {
      message.error(t('init.usernameExists'))
    } else if (errorMsg.includes('network') || error?.code === 'ECONNREFUSED') {
      message.error(t('init.networkError'))
    } else {
      message.error(errorMsg)
    }
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  const initialized = await authStore.checkInitialized()
  if (initialized) {
    router.push('/login')
  }
})
</script>

<style scoped>
.init-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
}

.init-card {
  width: 100%;
  max-width: 400px;
  padding: 40px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.init-header {
  text-align: center;
  margin-bottom: 32px;
}

.init-header .n-icon {
  color: #18a058;
  margin-bottom: 16px;
}

.init-header h1 {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  margin: 0 0 8px;
}

.init-header p {
  font-size: 14px;
  color: #999;
  margin: 0;
}

.init-success {
  text-align: center;
  padding: 32px 0;
}

.init-success p {
  margin: 16px 0 0;
  font-size: 16px;
  color: #333;
}

.redirect-text {
  color: #999;
  font-size: 14px !important;
}
</style>
