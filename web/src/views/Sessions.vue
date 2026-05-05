<template>
  <div class="sessions-page max-w-5xl">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-semibold">{{ t('sessions.title') }}</h2>
      <Button v-if="data.length > 0" variant="destructive" @click="showRevokeAllModal = true">
        <XCircle class="h-4 w-4 mr-2" />
        {{ t('sessions.revokeAll') }}
      </Button>
    </div>

    <Card>
      <CardHeader>
        <div class="flex items-center gap-2">
          <Clock class="h-4 w-4" />
          <span>{{ t('sessions.activeSessions', { count: data.length }) }}</span>
        </div>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('sessions.ipAddress') }}</TableHead>
              <TableHead>{{ t('sessions.userAgent') }}</TableHead>
              <TableHead>{{ t('sessions.created') }}</TableHead>
              <TableHead>{{ t('sessions.expires') }}</TableHead>
              <TableHead>{{ t('sessions.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in data" :key="row.id">
              <TableCell>{{ row.ip }}</TableCell>
              <TableCell class="max-w-[200px] truncate">{{ row.user_agent }}</TableCell>
              <TableCell>{{ new Date(row.created_at).toLocaleString() }}</TableCell>
              <TableCell>
                <Badge :variant="isExpiringSoon(row.expires_at) ? 'warning' : 'default'">
                  {{ new Date(row.expires_at).toLocaleString() }}
                </Badge>
              </TableCell>
              <TableCell>
                <AlertDialog>
                  <AlertDialogTrigger as-child>
                    <Button variant="ghost" size="sm">
                      <LogOut class="h-4 w-4" />
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>{{ t('sessions.revokeConfirm') }}</AlertDialogTitle>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>{{ t('sessions.cancel') }}</AlertDialogCancel>
                      <AlertDialogAction @click="handleRevoke(row.id)">{{ t('sessions.revoke') }}</AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <AlertDialog v-model:open="showRevokeAllModal">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{{ t('sessions.revokeAll') }}</AlertDialogTitle>
        </AlertDialogHeader>
        <div class="space-y-2">
          <p>{{ t('sessions.revokeAllConfirm') }}</p>
          <p class="text-sm text-muted-foreground">{{ t('sessions.revokeAllWarning') }}</p>
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel>{{ t('sessions.cancel') }}</AlertDialogCancel>
          <AlertDialogAction @click="handleRevokeAll">{{ t('sessions.revokeAll') }}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { XCircle, Clock, LogOut } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import {Card, CardHeader, CardTitle, CardContent} from '@/components/ui'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui'
import Badge from '@/components/ui/Badge.vue'
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from '@/components/ui'
import { useToast } from '@/components/ui/useToast.ts'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const { t } = useI18n()
const { toast } = useToast()
const router = useRouter()
const authStore = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const data = ref<any[]>([])
const showRevokeAllModal = ref(false)

function isExpiringSoon(expires_at: string) {
  const expires = new Date(expires_at)
  const now = new Date()
  return expires.getTime() - now.getTime() < 3600000
}

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/sessions')
    data.value = res.data || []
  } catch {
    toast({ title: 'Error', description: t('sessions.failedToLoad'), variant: 'destructive' })
  } finally {
    loading.value = false
  }
}

async function handleRevoke(id: number) {
  try {
    await api.delete(`/sessions/${id}`)
    toast({ title: t('sessions.sessionRevoked') })
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('sessions.failedToRevoke'), variant: 'destructive' })
  }
}

async function handleRevokeAll() {
  saving.value = true
  try {
    await api.delete('/sessions')
    toast({ title: t('sessions.allSessionsRevoked') })
    showRevokeAllModal.value = false
    authStore.logout()
    router.push('/login')
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('sessions.failedToRevokeAll'), variant: 'destructive' })
    saving.value = false
  }
}

onMounted(loadData)
</script>