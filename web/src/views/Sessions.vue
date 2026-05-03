<template>
  <div class="sessions-page">
    <div class="page-header">
      <h2>{{ t('sessions.title') }}</h2>
      <n-button type="error" @click="showRevokeAllModal = true" v-if="data.length > 0">
        <template #icon>
          <n-icon><close-circle-outline /></n-icon>
        </template>
        {{ t('sessions.revokeAll') }}
      </n-button>
    </div>

    <n-card bordered>
      <template #header>
        <n-space>
          <n-icon><time-outline /></n-icon>
          <span>{{ t('sessions.activeSessions', { count: data.length }) }}</span>
        </n-space>
      </template>

      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :row-key="(row: any) => row.id"
        :pagination="false"
      />
    </n-card>

    <n-modal v-model:show="showRevokeAllModal" preset="card" :title="t('sessions.revokeAll')" style="width: 400px">
      <n-space vertical>
        <n-text>{{ t('sessions.revokeAllConfirm') }}</n-text>
        <n-text depth="3">{{ t('sessions.revokeAllWarning') }}</n-text>
      </n-space>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showRevokeAllModal = false">{{ t('sessions.cancel') }}</n-button>
          <n-button type="error" @click="handleRevokeAll" :loading="saving">{{ t('sessions.revokeAll') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NDataTable, NCard, NButton, NSpace, NIcon, NTag, NModal, NText,
  NPopconfirm, useMessage
} from 'naive-ui'
import {
  TimeOutline, CloseCircleOutline, LogOutOutline
} from '@vicons/ionicons5'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'

const { t } = useI18n()
const message = useMessage()
const router = useRouter()
const authStore = useAuthStore()

const loading = ref(true)
const saving = ref(false)
const data = ref<any[]>([])
const showRevokeAllModal = ref(false)

const columns = [
  {
    title: () => t('sessions.ipAddress'),
    key: 'ip',
    width: 150
  },
  {
    title: () => t('sessions.userAgent'),
    key: 'user_agent',
    ellipsis: { tooltip: true }
  },
  {
    title: () => t('sessions.created'),
    key: 'created_at',
    width: 180,
    render: (row: any) => new Date(row.created_at).toLocaleString()
  },
  {
    title: () => t('sessions.expires'),
    key: 'expires_at',
    width: 180,
    render: (row: any) => {
      const expires = new Date(row.expires_at)
      const now = new Date()
      const isExpiringSoon = expires.getTime() - now.getTime() < 3600000

      return h(NTag, {
        type: isExpiringSoon ? 'warning' : 'default',
        size: 'small'
      }, {
        default: () => expires.toLocaleString()
      })
    }
  },
  {
    title: () => t('sessions.actions'),
    key: 'actions',
    width: 120,
    render: (row: any) => h(NPopconfirm, {
      onPositiveClick: () => handleRevoke(row.id)
    }, {
      trigger: () => h(NButton, {
        size: 'small',
        type: 'error',
        quaternary: true
      }, {
        icon: () => h(NIcon, null, { default: () => h(LogOutOutline) }),
        default: () => t('sessions.revoke')
      }),
      default: () => t('sessions.revokeConfirm')
    })
  }
]

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/sessions')
    data.value = res.data || []
  } catch {
    message.error(t('sessions.failedToLoad'))
  } finally {
    loading.value = false
  }
}

async function handleRevoke(id: number) {
  try {
    await api.delete(`/sessions/${id}`)
    message.success(t('sessions.sessionRevoked'))
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('sessions.failedToRevoke'))
  }
}

async function handleRevokeAll() {
  saving.value = true
  try {
    await api.delete('/sessions')
    message.success(t('sessions.allSessionsRevoked'))
    showRevokeAllModal.value = false
    authStore.logout()
    router.push('/login')
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('sessions.failedToRevokeAll'))
    saving.value = false
  }
}

onMounted(loadData)
</script>

<style scoped>
.sessions-page {
  max-width: 1000px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-header h2 {
  margin: 0;
}
</style>