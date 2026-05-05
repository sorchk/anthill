<template>
  <div class="flex min-h-screen items-center justify-center bg-gray-100">
    <div class="w-full max-w-[400px] p-8 bg-white rounded-lg shadow-lg">
      <div class="text-center mb-8">
        <Server class="h-12 w-12 mx-auto mb-4 text-primary" />
        <h1 class="text-2xl font-semibold">{{ t('login.title') }}</h1>
        <p class="text-sm text-muted-foreground mt-1">{{ t('login.subtitle') }}</p>
      </div>

      <form @submit.prevent="handleLogin" class="space-y-4">
        <div class="space-y-2">
          <Label for="username">{{ t('login.username') }}</Label>
          <Input
            id="username"
            v-model="formValue.username"
            :placeholder="t('login.username')"
            :maxlength="32"
            autocomplete="username"
          />
        </div>

        <div class="space-y-2">
          <Label for="password">{{ t('login.password') }}</Label>
          <Input
            id="password"
            v-model="formValue.password"
            type="password"
            :placeholder="t('login.password')"
            :maxlength="64"
            autocomplete="current-password"
          />
        </div>

        <Button type="submit" class="w-full" :disabled="loading">
          <Loader2 v-if="loading" class="mr-2 h-4 w-4 animate-spin" />
          {{ loading ? t('login.loggingIn') : t('login.signIn') }}
        </Button>
      </form>

      <div class="mt-6 text-center">
        <p class="text-xs text-muted-foreground">{{ t('login.defaultCredentials') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Server, Loader2 } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import { useToast } from '@/components/ui/useToast.ts'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const { t } = useI18n()
const { toast } = useToast()
const authStore = useAuthStore()

const loading = ref(false)

const formValue = ref({
  username: '',
  password: ''
})

async function handleLogin() {
  if (loading.value) return
  if (!formValue.value.username || !formValue.value.password) return

  loading.value = true

  try {
    await authStore.login(formValue.value.username, formValue.value.password)

    toast({
      title: t('login.welcome'),
      description: t('login.loggedInAs', { username: formValue.value.username }),
      variant: 'default'
    })

    router.push('/')
  } catch (error: any) {
    toast({
      title: 'Error',
      description: t('login.loginFailed'),
      variant: 'destructive'
    })
  } finally {
    loading.value = false
  }
}
</script>