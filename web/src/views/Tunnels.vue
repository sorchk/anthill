<template>
  <div class="tunnels-page">
    <div class="page-header">
      <h2>{{ t('tunnels.title') }}</h2>
      <n-button type="primary" @click="showAddModal = true">
        <template #icon>
          <n-icon><add-outline /></n-icon>
        </template>
        {{ t('tunnels.addTunnel') }}
      </n-button>
    </div>

    <n-card bordered>
      <template #header>
        <n-space>
          <span>{{ t('tunnels.total', { count: data.length }) }}</span>
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

    <n-modal v-model:show="showAddModal" preset="card" :title="t('tunnels.addTunnel')" style="width: 600px">
      <n-form :model="formValue" :rules="rules" ref="formRef" label-placement="top">
        <n-form-item :label="t('tunnels.name')" path="name">
          <n-input v-model:value="formValue.name" placeholder="My Tunnel" />
        </n-form-item>
        <n-form-item :label="t('tunnels.type')" path="type">
          <n-select
            v-model:value="formValue.type"
            :options="typeOptions"
            :placeholder="t('tunnels.selectType')"
          />
        </n-form-item>
        <n-form-item :label="t('tunnels.node')" path="node_id">
          <n-select
            v-model:value="formValue.node_id"
            :options="nodeOptions"
            :placeholder="t('tunnels.selectNode')"
            :loading="loadingNodes"
          />
        </n-form-item>
        <n-form-item :label="t('tunnels.transportMode')" path="transport_mode">
          <n-select
            v-model:value="formValue.transport_mode"
            :options="transportModeOptions"
            :placeholder="t('tunnels.selectTransportMode')"
          />
        </n-form-item>
        <n-form-item :label="t('tunnels.localAddr')" path="local_addr">
          <n-input v-model:value="formValue.local_addr" placeholder=":8080" />
        </n-form-item>
        <n-form-item :label="t('tunnels.remoteAddr')" path="remote_addr">
          <n-input v-model:value="formValue.remote_addr" placeholder="10.0.0.1:80" />
        </n-form-item>
        <n-form-item :label="t('tunnels.obfuscationMode')" path="obfuscation_mode">
          <n-select
            v-model:value="formValue.obfuscation_mode"
            :options="obfuscationModeOptions"
            :placeholder="t('tunnels.selectObfuscationMode')"
          />
        </n-form-item>
        <n-form-item :label="t('tunnels.e2eEnabled')" path="e2e_enabled">
          <n-switch v-model:value="formValue.e2e_enabled" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAddModal = false">{{ t('tunnels.cancel') }}</n-button>
          <n-button type="primary" @click="handleAdd" :loading="saving">{{ t('tunnels.add') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <n-modal v-model:show="showEditModal" preset="card" :title="t('tunnels.editTunnel')" style="width: 600px">
      <n-form :model="editFormValue" :rules="rules" ref="editFormRef" label-placement="top">
        <n-form-item :label="t('tunnels.name')" path="name">
          <n-input v-model:value="editFormValue.name" placeholder="My Tunnel" />
        </n-form-item>
        <n-form-item :label="t('tunnels.type')" path="type">
          <n-select
            v-model:value="editFormValue.type"
            :options="typeOptions"
            :placeholder="t('tunnels.selectType')"
          />
        </n-form-item>
        <n-form-item :label="t('tunnels.transportMode')" path="transport_mode">
          <n-select
            v-model:value="editFormValue.transport_mode"
            :options="transportModeOptions"
            :placeholder="t('tunnels.selectTransportMode')"
          />
        </n-form-item>
        <n-form-item :label="t('tunnels.localAddr')" path="local_addr">
          <n-input v-model:value="editFormValue.local_addr" placeholder=":8080" />
        </n-form-item>
        <n-form-item :label="t('tunnels.remoteAddr')" path="remote_addr">
          <n-input v-model:value="editFormValue.remote_addr" placeholder="10.0.0.1:80" />
        </n-form-item>
        <n-form-item :label="t('tunnels.obfuscationMode')" path="obfuscation_mode">
          <n-select
            v-model:value="editFormValue.obfuscation_mode"
            :options="obfuscationModeOptions"
            :placeholder="t('tunnels.selectObfuscationMode')"
          />
        </n-form-item>
        <n-form-item :label="t('tunnels.e2eEnabled')" path="e2e_enabled">
          <n-switch v-model:value="editFormValue.e2e_enabled" />
        </n-form-item>
        <n-form-item :label="t('tunnels.status')" path="status">
          <n-select
            v-model:value="editFormValue.status"
            :options="statusOptions"
            :placeholder="t('tunnels.selectStatus')"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditModal = false">{{ t('tunnels.cancel') }}</n-button>
          <n-button type="primary" @click="handleEdit" :loading="saving">{{ t('tunnels.save') }}</n-button>
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
  NFormItem, NInput, NSelect, NSwitch, NTag, NPopconfirm, useMessage
} from 'naive-ui'
import { AddOutline, RefreshOutline, TrashOutline, PencilOutline, PlayOutline, PauseOutline } from '@vicons/ionicons5'
import api from '@/api'
import { tunnelApi, type Tunnel, type CreateTunnelRequest, type UpdateTunnelRequest } from '@/api/tunnel'

const { t } = useI18n()
const message = useMessage()
const loading = ref(true)
const saving = ref(false)
const loadingNodes = ref(false)
const data = ref<Tunnel[]>([])
const nodes = ref<any[]>([])
const showAddModal = ref(false)
const showEditModal = ref(false)
const formRef = ref()
const editFormRef = ref()
const editingId = ref<number | null>(null)

const formValue = ref<CreateTunnelRequest>({
  name: '',
  type: 'port_forward',
  transport_mode: 'auto',
  local_addr: '',
  remote_addr: '',
  obfuscation_mode: 'none',
  e2e_enabled: true,
  node_id: ''
})

const editFormValue = ref<UpdateTunnelRequest & { id?: number }>({
  name: '',
  type: '',
  transport_mode: '',
  local_addr: '',
  remote_addr: '',
  obfuscation_mode: '',
  e2e_enabled: true,
  status: ''
})

const rules = {
  name: { required: true, message: t('tunnels.name') + ' is required' },
  type: { required: true, message: t('tunnels.type') + ' is required' },
  node_id: { required: true, message: t('tunnels.node') + ' is required' }
}

const typeOptions = [
  { label: 'Port Forward', value: 'port_forward' },
  { label: 'SOCKS5', value: 'socks5' },
  { label: 'HTTP Proxy', value: 'http' }
]

const transportModeOptions = [
  { label: 'Auto', value: 'auto' },
  { label: 'Direct', value: 'direct' },
  { label: 'Relay', value: 'relay' }
]

const obfuscationModeOptions = [
  { label: 'None', value: 'none' },
  { label: 'HTTP/2 Masquerade', value: 'http2_masquerade' },
  { label: 'Domain Fronting', value: 'domain_fronting' },
  { label: 'Traffic Padding', value: 'traffic_padding' }
]

const statusOptions = [
  { label: 'Active', value: 'active' },
  { label: 'Paused', value: 'paused' },
  { label: 'Stopped', value: 'stopped' }
]

const nodeOptions = ref<{ label: string; value: string }[]>([])

const columns = [
  { title: () => t('tunnels.name'), key: 'name', width: 150 },
  { title: () => t('tunnels.type'), key: 'type', width: 120 },
  {
    title: () => t('tunnels.status'),
    key: 'status',
    width: 100,
    render: (row: any) => {
      const type = row.status === 'active' ? 'success' : row.status === 'paused' ? 'warning' : 'default'
      return h(NTag, { type, size: 'small' }, { default: () => row.status || 'unknown' })
    }
  },
  { title: () => t('tunnels.transportMode'), key: 'transport_mode', width: 120 },
  { title: () => t('tunnels.localAddr'), key: 'local_addr', width: 120 },
  { title: () => t('tunnels.remoteAddr'), key: 'remote_addr', width: 150 },
  {
    title: () => t('tunnels.e2eEnabled'),
    key: 'e2e_enabled',
    width: 100,
    render: (row: any) => h(NTag, { type: row.e2e_enabled ? 'success' : 'default', size: 'small' }, {
      default: () => row.e2e_enabled ? 'Yes' : 'No'
    })
  },
  {
    title: () => t('tunnels.bytesIn'),
    key: 'bytes_in',
    width: 100,
    render: (row: any) => formatBytes(row.bytes_in || 0)
  },
  {
    title: () => t('tunnels.bytesOut'),
    key: 'bytes_out',
    width: 100,
    render: (row: any) => formatBytes(row.bytes_out || 0)
  },
  {
    title: () => t('tunnels.actions'),
    key: 'actions',
    width: 150,
    render: (row: any) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, {
          size: 'small',
          quaternary: true,
          onClick: () => handleToggle(row.id),
          title: row.status === 'active' ? t('tunnels.pause') : t('tunnels.resume')
        }, { icon: () => h(NIcon, null, { default: () => row.status === 'active' ? h(PauseOutline) : h(PlayOutline) }) }),
        h(NButton, {
          size: 'small',
          quaternary: true,
          onClick: () => openEditModal(row),
          title: t('tunnels.edit')
        }, { icon: () => h(NIcon, null, { default: () => h(PencilOutline) }) }),
        h(NPopconfirm, {
          onPositiveClick: () => handleDelete(row.id)
        }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, circle: true },
            { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
          default: () => t('tunnels.deleteConfirm')
        })
      ]
    })
  }
]

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

async function loadData() {
  loading.value = true
  try {
    data.value = await tunnelApi.list()
  } catch (e) {
    message.error(t('tunnels.failedToLoad'))
  } finally {
    loading.value = false
  }
}

async function loadNodes() {
  loadingNodes.value = true
  try {
    const res = await api.get('/nodes')
    nodes.value = res.data || []
    nodeOptions.value = nodes.value.map((n: any) => ({
      label: n.name,
      value: String(n.id)
    }))
  } catch (e) {
    console.error('Failed to load nodes', e)
  } finally {
    loadingNodes.value = false
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
    await tunnelApi.create(formValue.value)
    message.success(t('tunnels.tunnelCreated'))
    showAddModal.value = false
    formValue.value = {
      name: '',
      type: 'port_forward',
      transport_mode: 'auto',
      local_addr: '',
      remote_addr: '',
      obfuscation_mode: 'none',
      e2e_enabled: true,
      node_id: ''
    }
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('tunnels.failedToCreate'))
  } finally {
    saving.value = false
  }
}

function openEditModal(tunnel: Tunnel) {
  editingId.value = tunnel.id || null
  editFormValue.value = {
    name: tunnel.name,
    type: tunnel.type,
    transport_mode: tunnel.transport_mode,
    local_addr: tunnel.local_addr,
    remote_addr: tunnel.remote_addr,
    obfuscation_mode: tunnel.obfuscation_mode,
    e2e_enabled: tunnel.e2e_enabled,
    status: tunnel.status
  }
  showEditModal.value = true
}

async function handleEdit() {
  if (editingId.value === null) return

  saving.value = true
  try {
    await tunnelApi.update(editingId.value, editFormValue.value)
    message.success(t('tunnels.tunnelUpdated'))
    showEditModal.value = false
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('tunnels.failedToUpdate'))
  } finally {
    saving.value = false
  }
}

async function handleToggle(id: number) {
  try {
    const res = await tunnelApi.toggle(id)
    message.success(res.status === 'active' ? t('tunnels.tunnelResumed') : t('tunnels.tunnelPaused'))
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('tunnels.failedToToggle'))
  }
}

async function handleDelete(id: number) {
  try {
    await tunnelApi.delete(id)
    message.success(t('tunnels.tunnelDeleted'))
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('tunnels.failedToDelete'))
  }
}

onMounted(() => {
  loadData()
  loadNodes()
})
</script>

<style scoped>
.tunnels-page {
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
</style>