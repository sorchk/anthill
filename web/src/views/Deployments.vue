<template>
  <div class="deployments-page">
    <div class="page-header">
      <h2>{{ t('deployments.title') }}</h2>
      <n-button type="primary" @click="showDeployModal = true">
        <template #icon>
          <n-icon><cloud-upload-outline /></n-icon>
        </template>
        {{ t('deployments.deployPlugin') }}
      </n-button>
    </div>

    <n-card bordered>
      <template #header>
        <n-space>
          <n-tag type="info">{{ t('deployments.total') }}: {{ data.length }}</n-tag>
          <n-tag type="success">{{ t('deployments.deployed') }}: {{ deployedCount }}</n-tag>
          <n-tag type="warning">{{ t('deployments.pending') }}: {{ pendingCount }}</n-tag>
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
        :pagination="{ pageSize: 20 }"
      />
    </n-card>

    <!-- Deploy Modal -->
    <n-modal v-model:show="showDeployModal" preset="card" :title="t('deployments.deployPlugin')" style="width: 500px">
      <n-form :model="deployForm" ref="deployFormRef" label-placement="top">
        <n-form-item :label="t('deployments.plugin')" path="plugin_id">
          <n-select
            v-model:value="deployForm.plugin_id"
            :options="pluginOptions"
            :placeholder="t('deployments.selectPlugin')"
            filterable
          />
        </n-form-item>
        <n-form-item :label="t('deployments.node')" path="node_ids">
          <n-select
            v-model:value="deployForm.node_ids"
            :options="nodeOptions"
            :placeholder="t('deployments.selectNodes')"
            multiple
            filterable
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showDeployModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" @click="handleDeploy" :loading="saving">{{ t('deployments.deployPlugin') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NDataTable, NCard, NButton, NSpace, NIcon, NModal, NForm,
  NFormItem, NSelect, NTag, useMessage, NPopconfirm
} from 'naive-ui'
import {
  CloudUploadOutline, RefreshOutline, CheckmarkCircle,
  TimeOutline, CloseCircle, StopOutline
} from '@vicons/ionicons5'
import api from '@/api'

const { t } = useI18n()
const message = useMessage()

const loading = ref(true)
const saving = ref(false)
const data = ref<any[]>([])
const nodes = ref<any[]>([])
const plugins = ref<any[]>([])
const showDeployModal = ref(false)

const deployFormRef = ref()
const deployForm = ref({ plugin_id: null, node_ids: [] })

const nodeOptions = computed(() =>
  nodes.value.map(n => ({ label: `${n.name} (${n.host})`, value: n.id }))
)

const pluginOptions = computed(() =>
  plugins.value.map(p => ({ label: `${p.name} - ${p.version}`, value: p.id }))
)

const deployedCount = computed(() => data.value.filter(d => d.status === 'deployed').length)
const pendingCount = computed(() => data.value.filter(d => d.status === 'pending' || d.status === 'in_progress').length)

const columns = [
  { title: 'ID', key: 'id', width: 80 },
  {
    title: () => t('deployments.plugin'),
    key: 'plugin_name',
    width: 180,
    ellipsis: { tooltip: true }
  },
  {
    title: () => t('deployments.node'),
    key: 'node_name',
    width: 150,
    ellipsis: { tooltip: true }
  },
  { title: () => t('deployments.version'), key: 'version', width: 100 },
  {
    title: () => t('deployments.status'),
    key: 'status',
    width: 120,
    render: (row: any) => {
      const config: Record<string, { type: 'success' | 'warning' | 'info' | 'error' | 'default' | 'primary'; icon: any }> = {
        deployed: { type: 'success', icon: CheckmarkCircle },
        pending: { type: 'warning', icon: TimeOutline },
        in_progress: { type: 'info', icon: TimeOutline },
        failed: { type: 'error', icon: CloseCircle },
        cancelled: { type: 'default', icon: StopOutline }
      }
      const { type, icon } = config[row.status] as { type: 'success' | 'warning' | 'info' | 'error' | 'default' | 'primary'; icon: any } || { type: 'default' as const, icon: TimeOutline }
      return h(NTag, { type, size: 'small' }, {
        icon: () => h(NIcon, null, { default: () => h(icon) }),
        default: () => row.status
      })
    }
  },
  {
    title: () => t('deployments.deployedBy'),
    key: 'deployed_by',
    width: 100
  },
  {
    title: () => t('deployments.created'),
    key: 'created_at',
    width: 180,
    render: (row: any) => new Date(row.created_at).toLocaleString()
  },
  {
    title: () => t('deployments.actions'),
    key: 'actions',
    width: 100,
    render: (row: any) => {
      if (row.status === 'pending') {
        return h(NPopconfirm, {
          onPositiveClick: () => handleCancel(row.id)
        }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true }, t('deployments.cancel')),
          default: () => t('deployments.cancelConfirm')
        })
      }
      return null
    }
  }
]

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/deployments')
    data.value = res.data || []
  } catch {
    message.error(t('deployments.failedToLoad'))
  } finally {
    loading.value = false
  }
}

async function loadNodesAndPlugins() {
  try {
    const [nodesRes, pluginsRes] = await Promise.all([
      api.get('/nodes'),
      api.get('/plugins')
    ])
    nodes.value = nodesRes.data || []
    plugins.value = pluginsRes.data || []
  } catch {
    message.error(t('deployments.failedToLoad'))
  }
}

async function handleDeploy() {
  if (!deployForm.value.plugin_id || deployForm.value.node_ids.length === 0) {
    message.warning(t('deployments.pleaseSelectNodes'))
    return
  }

  saving.value = true
  try {
    await api.post('/deployments', {
      plugin_id: deployForm.value.plugin_id,
      node_ids: deployForm.value.node_ids
    })
    message.success(t('deployments.deploymentInitiated'))
    showDeployModal.value = false
    deployForm.value = { plugin_id: null, node_ids: [] }
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('deployments.failedToDeploy'))
  } finally {
    saving.value = false
  }
}

async function handleCancel(id: number) {
  try {
    await api.post(`/deployments/${id}/cancel`)
    message.success(t('deployments.deploymentCancelled'))
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('deployments.failedToCancel'))
  }
}

onMounted(async () => {
  await Promise.all([loadData(), loadNodesAndPlugins()])
})
</script>

<style scoped>
.deployments-page {
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