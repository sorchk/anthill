<template>
  <div class="settings-page max-w-4xl">
    <h2 class="text-2xl font-semibold mb-6">{{ t('settings.title') }}</h2>

    <div class="space-y-6">
      <Tabs v-model:value="activeTab">
        <TabsList>
          <TabsTrigger value="profile">{{ t('settings.profile') }}</TabsTrigger>
          <TabsTrigger value="password">{{ t('settings.changePassword') }}</TabsTrigger>
          <TabsTrigger value="appearance">{{ t('settings.appearance') }}</TabsTrigger>
          <TabsTrigger v-if="authStore.isAdmin" value="system">{{ t('settings.system') }}</TabsTrigger>
        </TabsList>

        <TabsContent value="profile">
          <Card>
            <CardHeader>
              <CardTitle>{{ t('settings.profileInfo') }}</CardTitle>
            </CardHeader>
            <CardContent>
              <form @submit.prevent="updateProfile" class="space-y-4">
                <div class="space-y-2">
                  <Label for="username">{{ t('users.username') }}</Label>
                  <Input id="username" v-model="profileForm.username" />
                </div>
                <div class="space-y-2">
                  <Label for="role">{{ t('users.role') }}</Label>
                  <Input id="role" :model-value="authStore.user?.role" disabled />
                </div>
                <Button type="submit" :disabled="savingProfile">
                  <Loader2 v-if="savingProfile" class="mr-2 h-4 w-4 animate-spin" />
                  {{ t('settings.save') }}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="password">
          <Card>
            <CardHeader>
              <CardTitle>{{ t('settings.changePassword') }}</CardTitle>
            </CardHeader>
            <CardContent>
              <form @submit.prevent="changePassword" class="space-y-4">
                <div class="space-y-2">
                  <Label for="old-password">{{ t('users.currentPassword') }}</Label>
                  <Input id="old-password" v-model="passwordForm.old_password" type="password" />
                </div>
                <div class="space-y-2">
                  <Label for="new-password">{{ t('users.newPassword') }}</Label>
                  <Input id="new-password" v-model="passwordForm.new_password" type="password" />
                </div>
                <div class="space-y-2">
                  <Label for="confirm-password">{{ t('settings.confirmPassword') }}</Label>
                  <Input id="confirm-password" v-model="passwordForm.confirm_password" type="password" />
                </div>
                <Button type="submit" :disabled="savingPassword">
                  <Loader2 v-if="savingPassword" class="mr-2 h-4 w-4 animate-spin" />
                  {{ t('settings.changePassword') }}
                </Button>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="appearance">
          <Card>
            <CardHeader>
              <CardTitle>{{ t('settings.appearance') }}</CardTitle>
            </CardHeader>
            <CardContent>
              <div class="space-y-4">
                <div class="space-y-2">
                  <Label for="language">{{ t('settings.language') }}</Label>
                  <Select v-model="appearanceForm.language" @update:modelValue="changeLanguage">
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="zh-CN">简体中文</SelectItem>
                      <SelectItem value="en-US">English</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent v-if="authStore.isAdmin" value="system">
          <div class="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>{{ t('settings.systemInfo') }}</CardTitle>
              </CardHeader>
              <CardContent>
                <div class="grid grid-cols-2 gap-4 text-sm">
                  <div><span class="text-muted-foreground">{{ t('settings.version') }}:</span> v1.0.0</div>
                  <div><span class="text-muted-foreground">{{ t('settings.database') }}:</span> SQLite</div>
                  <div><span class="text-muted-foreground">{{ t('settings.goVersion') }}:</span> Go 1.21+</div>
                  <div><span class="text-muted-foreground">{{ t('settings.frameworks') }}:</span> Gin + Vue3</div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{{ t('settings.security') }}</CardTitle>
              </CardHeader>
              <CardContent>
                <div class="space-y-2 text-sm">
                  <div>{{ t('settings.sessionTimeout') }}: 24h</div>
                  <div>{{ t('settings.csrfProtection') }}: {{ t('common.enabled') }}</div>
                  <div>{{ t('settings.tlsVersion') }}: TLS 1.3</div>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loader2 } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import {Card, CardHeader, CardTitle, CardContent} from '@/components/ui'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui'
import { useToast } from '@/components/ui/useToast.ts'
import { useAuthStore } from '@/stores/auth'
import api from '@/api'

const { t, locale } = useI18n()
const { toast } = useToast()
const authStore = useAuthStore()

const activeTab = ref('profile')
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

async function updateProfile() {
  if (!profileForm.value.username) return

  savingProfile.value = true
  try {
    await api.put(`/users/${authStore.user?.id}`, { username: profileForm.value.username })
    authStore.user && (authStore.user.username = profileForm.value.username)
    toast({ title: t('settings.profileUpdated') })
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('settings.updateFailed'), variant: 'destructive' })
  } finally {
    savingProfile.value = false
  }
}

async function changePassword() {
  if (passwordForm.value.new_password !== passwordForm.value.confirm_password) {
    toast({ title: 'Error', description: t('settings.passwordMismatch'), variant: 'destructive' })
    return
  }

  savingPassword.value = true
  try {
    await api.post(`/users/${authStore.user?.id}/password`, {
      old_password: passwordForm.value.old_password,
      new_password: passwordForm.value.new_password
    })
    toast({ title: t('settings.passwordChanged') })
    passwordForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('settings.passwordChangeFailed'), variant: 'destructive' })
  } finally {
    savingPassword.value = false
  }
}

function changeLanguage(lang: string) {
  locale.value = lang
  localStorage.setItem('locale', lang)
}
</script>