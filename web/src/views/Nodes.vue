<template>
  <div class="nodes-page">
    <div class="page-header">
      <h2>{{ t('nodes.title') }}</h2>
      <n-button type="primary" @click="showAddModal = true">
        <template #icon>
          <n-icon><add-outline /></n-icon>
        </template>
        {{ t('nodes.addNode') }}
      </n-button>
    </div>

    <n-card bordered>
      <template #header>
        <n-space>
          <span>{{ t('nodes.total', { count: data.length }) }}</span>
          <n-button size="small" @click="loadData">
            <template #icon><n-icon><refresh-outline /></n-icon></template>
          </n-button>
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

    <n-modal v-model:show="showAddModal" preset="card" :title="t('nodes.addNode')" style="width: 600px">
      <n-form :model="formValue" :rules="rules" ref="formRef" label-placement="top">
        <n-form-item :label="t('nodes.name')" path="name">
          <n-input v-model:value="formValue.name" placeholder="My Node" />
        </n-form-item>
        <n-form-item :label="t('nodes.host')" path="host">
          <n-input v-model:value="formValue.host" placeholder="192.168.1.100" />
        </n-form-item>
        <n-form-item :label="t('nodes.port')" path="port">
          <n-input-number v-model:value="formValue.port" :min="1" :max="65535" style="width: 100%" />
        </n-form-item>
        <n-form-item :label="t('nodes.connectMode')">
          <n-select
            v-model:value="formValue.connect_mode"
            :options="connectModeOptions"
            :placeholder="t('nodes.selectConnectMode')"
          />
        </n-form-item>
        <n-form-item :label="t('nodes.nodeHost')">
          <n-input v-model:value="formValue.node_host" :placeholder="t('nodes.nodeHostPlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('nodes.nodePort')">
          <n-input-number v-model:value="formValue.node_port" :min="1" :max="65535" style="width: 100%" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAddModal = false">{{ t('nodes.cancel') }}</n-button>
          <n-button type="primary" @click="handleAdd" :loading="saving">{{ t('nodes.addNode') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal v-model:show="showEditModal" preset="card" :title="t('nodes.edit')" style="width: 600px">
      <n-form :model="editForm" label-placement="top">
        <n-form-item :label="t('nodes.name')">
          <n-input v-model:value="editForm.name" />
        </n-form-item>
        <n-form-item :label="t('nodes.host')">
          <n-input v-model:value="editForm.host" />
        </n-form-item>
        <n-form-item :label="t('nodes.port')">
          <n-input-number v-model:value="editForm.port" :min="1" :max="65535" style="width: 100%" />
        </n-form-item>
        <n-form-item :label="t('nodes.connectMode')">
          <n-select
            v-model:value="editForm.connect_mode"
            :options="connectModeOptions"
            :placeholder="t('nodes.selectConnectMode')"
          />
        </n-form-item>
        <n-form-item :label="t('nodes.nodeHost')">
          <n-input v-model:value="editForm.node_host" :placeholder="t('nodes.nodeHostPlaceholder')" />
        </n-form-item>
        <n-form-item :label="t('nodes.nodePort')">
          <n-input-number v-model:value="editForm.node_port" :min="1" :max="65535" style="width: 100%" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditModal = false">{{ t('nodes.cancel') }}</n-button>
          <n-button type="primary" @click="handleUpdate" :loading="saving">{{ t('nodes.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NDataTable, NCard, NButton, NSpace, NIcon, NModal, NForm,
  NFormItem, NInput, NInputNumber, NTag, NPopconfirm, NSelect, useMessage
} from 'naive-ui'
import { AddOutline, RefreshOutline, TrashOutline, FlashOutline } from '@vicons/ionicons5'
import api from '@/api'

const { t } = useI18n()
const message = useMessage()
const loading = ref(true)
const saving = ref(false)
const data = ref<any[]>([])
const showAddModal = ref(false)
const showEditModal = ref(false)
const formRef = ref()
const editingId = ref<number | null>(null)

const formValue = ref({ name: '', host: '', port: 18888, connect_mode: 'passive_tls', node_host: '', node_port: 18888 })
const editForm = ref({ name: '', host: '', port: 18888, connect_mode: 'passive_tls', node_host: '', node_port: 18888 })

const connectModeOptions = [
  { label: 'Active TLS (admin connects to node via TLS)', value: 'active_tls' },
  { label: 'Active WSS (admin connects to node via WSS)', value: 'active_wss' },
  { label: 'Passive TLS (node connects to admin via TLS)', value: 'passive_tls' },
  { label: 'Passive WSS (node connects to admin via WSS)', value: 'passive_wss' },
  { label: 'Auto (try active, fallback to passive)', value: 'auto' }
]

const rules = {
  name: { required: true, message: t('nodes.name') + ' is required' },
  host: { required: true, message: t('nodes.host') + ' is required' },
  port: { required: true, type: 'number' as const, message: t('nodes.port') + ' is required' }
}

const columns = [
  { title: () => t('nodes.name'), key: 'name', width: 150 },
  { title: () => t('nodes.host'), key: 'host', width: 150 },
  { title: () => t('nodes.port'), key: 'port', width: 100 },
  {
    title: () => t('nodes.connectMode'),
    key: 'connect_mode',
    width: 130,
    render: (row: any) => h(NTag, { type: row.connect_mode?.startsWith('active') ? 'info' : 'default', size: 'small' },
      { default: () => row.connect_mode || 'passive_tls' })
  },
  {
    title: () => t('nodes.status'),
    key: 'status',
    width: 100,
    render: (row: any) => {
      const type = row.status === 'online' ? 'success' : row.status === 'offline' ? 'error' : 'default'
      return h(NTag, { type, size: 'small' }, { default: () => row.status || 'unknown' })
    }
  },
  {
    title: () => t('nodes.lastSeen'),
    key: 'last_seen',
    width: 180,
    render: (row: any) => row.last_seen ? new Date(row.last_seen).toLocaleString() : '-'
  },
  {
    title: () => t('nodes.actions'),
    key: 'actions',
    width: 150,
    render: (row: any) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, {
          size: 'small',
          quaternary: true,
          onClick: () => handleEdit(row)
        }, { default: () => t('nodes.edit') }),
        h(NButton, {
          size: 'small',
          quaternary: true,
          onClick: () => handleConnect(row.id),
          title: t('nodes.connect')
        }, { icon: () => h(NIcon, null, { default: () => h(FlashOutline) }) }),
        h(NPopconfirm, {
          onPositiveClick: () => handleDelete(row.id)
        }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, circle: true },
            { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
          default: () => t('nodes.deleteConfirm')
        })
      ]
    })
  }
]

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/nodes')
    data.value = res.data || []
  } catch (e) {
    message.error(t('nodes.failedToLoad'))
  } finally {
    loading.value = false
  }
}

async function handleAdd() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    await api.post('/nodes', formValue.value)
    message.success(t('nodes.nodeAdded'))
    showAddModal.value = false
    formValue.value = { name: '', host: '', port: 18888, connect_mode: 'passive_tls', node_host: '', node_port: 18888 }
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToAdd'))
  } finally {
    saving.value = false
  }
}

function handleEdit(row: any) {
  editingId.value = row.id
  editForm.value = {
    name: row.name,
    host: row.host,
    port: row.port,
    connect_mode: row.connect_mode || 'passive_tls',
    node_host: row.node_host || '',
    node_port: row.node_port || 18888
  }
  showEditModal.value = true
}

async function handleUpdate() {
  if (!editingId.value) return

  saving.value = true
  try {
    await api.put(`/nodes/${editingId.value}`, editForm.value)
    message.success(t('nodes.nodeUpdated'))
    showEditModal.value = false
    editingId.value = null
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToUpdate'))
  } finally {
    saving.value = false
  }
}

async function handleConnect(id: number) {
  try {
    const res = await api.post(`/nodes/${id}/connect`)
    message.success(res.data?.message || t('nodes.connectionInitiated'))
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToConnect'))
  }
}

async function handleDelete(id: number) {
  try {
    await api.delete(`/nodes/${id}`)
    message.success(t('nodes.nodeDeleted'))
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToDelete'))
  }
}

onMounted(loadData)
</script>

<style scoped>
.nodes-page {
  max-width: 1200px;
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