<template>
  <div class="audit-page">
    <div class="page-header">
      <h2>{{ t('audit.title') }}</h2>
      <n-space>
        <n-button @click="handleExport">
          <template #icon><n-icon><download-outline /></n-icon></template>
          {{ t('audit.export') }}
        </n-button>
        <n-select
          v-model:value="filters.action"
          :placeholder="t('audit.filterByAction')"
          :options="actionOptions"
          clearable
          style="width: 200px"
          @update:value="loadData"
        />
        <n-select
          v-model:value="filters.username"
          :placeholder="t('audit.filterByUser')"
          :options="userOptions"
          clearable
          filterable
          style="width: 150px"
          @update:value="loadData"
        />
        <n-button @click="loadData">
          <template #icon><n-icon><refresh-outline /></n-icon></template>
        </n-button>
      </n-space>
    </div>

    <n-card bordered>
      <n-data-table
        :columns="columns"
        :data="data"
        :loading="loading"
        :row-key="(row: any) => row.id"
        :pagination="pagination"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- Log Detail Modal -->
    <n-modal v-model:show="showDetailModal" preset="card" :title="`Audit Log #${selectedLog?.id}`" style="width: 600px">
      <n-descriptions v-if="selectedLog" :column="1" bordered size="small">
        <n-descriptions-item :label="t('audit.id')">{{ selectedLog.id }}</n-descriptions-item>
        <n-descriptions-item :label="t('audit.user')">{{ selectedLog.username }} ({{ selectedLog.user_id }})</n-descriptions-item>
        <n-descriptions-item :label="t('audit.action')">{{ selectedLog.action }}</n-descriptions-item>
        <n-descriptions-item :label="t('audit.method')">
          <n-tag :type="getMethodType(selectedLog.method) as any" size="small">{{ selectedLog.method }}</n-tag>
        </n-descriptions-item>
        <n-descriptions-item :label="t('audit.path')">{{ selectedLog.path }}</n-descriptions-item>
        <n-descriptions-item :label="t('audit.ip')">{{ selectedLog.ip }}</n-descriptions-item>
        <n-descriptions-item :label="t('audit.status')">
          <n-tag :type="selectedLog.status < 400 ? 'success' : 'error'" size="small">
            {{ selectedLog.status }}
          </n-tag>
        </n-descriptions-item>
        <n-descriptions-item :label="t('audit.time')">{{ formatDate(selectedLog.created_at) }}</n-descriptions-item>
        <n-descriptions-item :label="t('audit.details')" v-if="selectedLog.details">
          <pre class="details-pre">{{ selectedLog.details }}</pre>
        </n-descriptions-item>
      </n-descriptions>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showDetailModal = false">{{ t('audit.close') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NDataTable, NCard, NButton, NSpace, NIcon, NTag, NModal,
  NDescriptions, NDescriptionsItem, NSelect, useMessage
} from 'naive-ui'
import { RefreshOutline, DownloadOutline } from '@vicons/ionicons5'
import api from '@/api'

const { t } = useI18n()
const message = useMessage()

const loading = ref(true)
const data = ref<any[]>([])
const selectedLog = ref<any>(null)
const showDetailModal = ref(false)
const filters = ref({ action: null, username: null })

const pagination = ref({
  page: 1,
  pageSize: 50,
  showSizePicker: true,
  pageSizes: [25, 50, 100, 200],
  onChange: (page: number) => { pagination.value.page = page },
  onUpdatePageSize: (pageSize: number) => {
    pagination.value.pageSize = pageSize
    pagination.value.page = 1
  }
})

const actionOptions = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'DELETE', value: 'DELETE' }
]

const userOptions = ref<any[]>([])

const columns = [
  { title: () => t('audit.id'), key: 'id', width: 80 },
  {
    title: () => t('audit.user'),
    key: 'username',
    width: 120,
    ellipsis: { tooltip: true }
  },
  {
    title: () => t('audit.action'),
    key: 'action',
    ellipsis: { tooltip: true }
  },
  {
    title: () => t('audit.method'),
    key: 'method',
    width: 100,
    render: (row: any) => h(NTag, { type: getMethodType(row.method) as any, size: 'small' }, { default: () => row.method })
  },
  {
    title: () => t('audit.path'),
    key: 'path',
    ellipsis: { tooltip: true }
  },
  {
    title: () => t('audit.ip'),
    key: 'ip',
    width: 140
  },
  {
    title: () => t('audit.status'),
    key: 'status',
    width: 100,
    render: (row: any) => h(NTag, {
      type: row.status < 400 ? 'success' : 'error',
      size: 'small'
    }, { default: () => row.status })
  },
  {
    title: () => t('audit.time'),
    key: 'created_at',
    width: 180,
    render: (row: any) => formatDate(row.created_at)
  },
  {
    title: '',
    key: 'detail',
    width: 60,
    render: (row: any) => h(NButton, {
      size: 'small',
      quaternary: true,
      onClick: () => openDetail(row)
    }, { default: () => t('audit.view') })
  }
]

function getMethodType(method: string): 'info' | 'success' | 'warning' | 'error' | 'default' {
  const types: Record<string, 'info' | 'success' | 'warning' | 'error' | 'default'> = {
    GET: 'info',
    POST: 'success',
    PUT: 'warning',
    DELETE: 'error'
  }
  return types[method] || 'default'
}

function formatDate(date: string) {
  return new Date(date).toLocaleString()
}

function openDetail(log: any) {
  selectedLog.value = log
  showDetailModal.value = true
}

function handleExport() {
  window.open(`/api/audit/export?format=csv`, '_blank')
}

async function loadData() {
  loading.value = true
  try {
    const params: any = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    }

    const res = await api.get('/audit', { params })
    data.value = res.data?.data || []

    const userSet = new Set<string>()
    data.value.forEach((log: any) => userSet.add(log.username))
    userOptions.value = Array.from(userSet).map(u => ({ label: u, value: u }))
  } catch {
    message.error(t('audit.failedToLoad'))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.value.page = page
  loadData()
}

function handlePageSizeChange(pageSize: number) {
  pagination.value.pageSize = pageSize
  pagination.value.page = 1
  loadData()
}

onMounted(loadData)
</script>

<style scoped>
.audit-page {
  max-width: 1400px;
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

.details-pre {
  background: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
  font-size: 12px;
  overflow-x: auto;
  max-height: 200px;
}
</style>