<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-100">
    <div class="w-full max-w-[400px] p-8 bg-white rounded-lg shadow-lg">
      <div v-if="step === 1" class="space-y-6">
        <div class="text-center">
          <Server class="h-12 w-12 mx-auto mb-4 text-primary" />
          <h1 class="text-2xl font-semibold">{{ t('init.welcomeTitle') }}</h1>
          <p class="text-sm text-muted-foreground mt-1">{{ t('init.welcomeSubtitle') }}</p>
        </div>

        <form @submit.prevent="handleNext" class="space-y-4">
          <div class="space-y-2">
            <Label for="username">{{ t('init.username') }}</Label>
            <Input
              id="username"
              v-model="formValue.username"
              :placeholder="t('init.username')"
              :maxlength="32"
            />
          </div>

          <div class="space-y-2">
            <Label for="password">{{ t('init.password') }}</Label>
            <Input
              id="password"
              v-model="formValue.password"
              type="password"
              :placeholder="t('init.password')"
              :maxlength="64"
            />
          </div>

          <div class="space-y-2">
            <Label for="confirmPassword">{{ t('init.confirmPassword') }}</Label>
            <Input
              id="confirmPassword"
              v-model="formValue.confirmPassword"
              type="password"
              :placeholder="t('init.confirmPassword')"
              :maxlength="64"
              @keydown.enter="handleNext"
            />
          </div>

          <Button type="submit" class="w-full" :disabled="loading">
            <Loader2 v-if="loading" class="mr-2 h-4 w-4 animate-spin" />
            {{ loading ? t('init.creating') : t('init.next') }}
          </Button>
        </form>
      </div>

      <div v-else-if="step === 2" class="text-center py-8">
        <CheckCircle class="h-16 w-16 mx-auto mb-4 text-green-500" />
        <p class="text-lg">{{ t('init.createSuccess') }}</p>
        <p class="text-sm text-muted-foreground mt-2">{{ t('init.redirecting') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Server, CheckCircle, Loader2 } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import { useToast } from '@/components/ui/useToast.ts'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const { t } = useI18n()
const { toast } = useToast()
const authStore = useAuthStore()

const step = ref(1)
const loading = ref(false)

const formValue = ref({
  username: '',
  password: '',
  confirmPassword: ''
})

async function handleNext() {
  if (loading.value) return
  if (!formValue.value.username || !formValue.value.password || !formValue.value.confirmPassword) return
  if (formValue.value.password !== formValue.value.confirmPassword) {
    toast({ title: 'Error', description: t('init.passwordMismatch'), variant: 'destructive' })
    return
  }
  if (formValue.value.password.length < 6) {
    toast({ title: 'Error', description: t('init.passwordMinLength'), variant: 'destructive' })
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
      toast({ title: 'Error', description: t('init.usernameExists'), variant: 'destructive' })
    } else if (errorMsg.includes('network') || error?.code === 'ECONNREFUSED') {
      toast({ title: 'Error', description: t('init.networkError'), variant: 'destructive' })
    } else {
      toast({ title: 'Error', description: errorMsg, variant: 'destructive' })
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