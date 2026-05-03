<template>
  <div class="node-detail-page">
    <div class="page-header">
      <n-space>
        <n-button @click="$router.push('/nodes')">
          <template #icon><n-icon><arrow-back-outline /></n-icon></template>
        </n-button>
        <h2>{{ node?.name || t('nodes.nodeDetail') }}</h2>
        <n-tag :type="statusType" size="large">{{ node?.status || 'unknown' }}</n-tag>
      </n-space>
      <n-space>
        <n-button @click="testConnection" :loading="testing">
          <template #icon><n-icon><flash-outline /></n-icon></template>
          {{ t('nodes.testConnection') }}
        </n-button>
        <n-button type="primary" @click="showEditModal = true">
          <template #icon><n-icon><create-outline /></n-icon></template>
          {{ t('nodes.edit') }}
        </n-button>
      </n-space>
    </div>

    <n-tabs type="line" animated>
      <n-tab-pane name="info" :tab="t('nodes.info')">
        <n-card>
          <n-descriptions :column="2" bordered size="small" v-if="node">
            <n-descriptions-item :label="t('nodes.name')">{{ node.name }}</n-descriptions-item>
            <n-descriptions-item :label="t('nodes.host')">{{ node.host }}</n-descriptions-item>
            <n-descriptions-item :label="t('nodes.port')">{{ node.port }}</n-descriptions-item>
            <n-descriptions-item :label="t('nodes.status')">
              <n-tag :type="statusType">{{ node.status }}</n-tag>
            </n-descriptions-item>
            <n-descriptions-item :label="t('nodes.lastSeen')">
              {{ node.last_seen ? new Date(node.last_seen).toLocaleString() : '-' }}
            </n-descriptions-item>
            <n-descriptions-item :label="t('nodes.created')">
              {{ node.created_at ? new Date(node.created_at).toLocaleString() : '-' }}
            </n-descriptions-item>
          </n-descriptions>
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="plugins" :tab="t('nodes.plugins')">
        <n-card>
          <template #header>
            <n-space justify="space-between" align="center">
              <span>{{ t('nodes.installedPlugins') }}: {{ nodePlugins.length }}</span>
              <n-button size="small" @click="showInstallPluginModal = true" type="primary">
                <template #icon><n-icon><add-outline /></n-icon></template>
                {{ t('nodes.installPlugin') }}
              </n-button>
            </n-space>
          </template>

          <n-data-table
            v-if="nodePlugins.length > 0"
            :columns="pluginColumns"
            :data="nodePlugins"
            :loading="loadingPlugins"
            :pagination="false"
          />
          <n-empty v-else :description="t('nodes.noPlugins')" />
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="clients" :tab="t('nodes.clients')">
        <n-card>
          <template #header>
            <n-space justify="space-between" align="center">
              <span>{{ t('nodes.clients') }}: {{ clients.length }}</span>
              <n-button size="small" @click="showAddClientModal = true" type="primary">
                <template #icon><n-icon><add-outline /></n-icon></template>
                {{ t('nodes.addClient') }}
              </n-button>
            </n-space>
          </template>

          <n-data-table
            v-if="clients.length > 0"
            :columns="clientColumns"
            :data="clients"
            :loading="loadingClients"
            :pagination="false"
          />
          <n-empty v-else :description="t('nodes.noClients')" />
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="services" :tab="t('nodes.services')">
        <n-card>
          <template #header>
            <n-space justify="space-between" align="center">
              <span>{{ t('nodes.servicePlugins') }}</span>
              <n-button size="small" @click="loadServices">
                <template #icon><n-icon><refresh-outline /></n-icon></template>
              </n-button>
            </n-space>
          </template>

          <n-list v-if="services.length > 0">
            <n-list-item v-for="svc in services" :key="svc.name">
              <n-thing>
                <template #header>
                  <n-space>
                    <span>{{ svc.name }}</span>
                    <n-tag :type="svc.status === 'running' ? 'success' : 'default'" size="small">
                      {{ svc.status }}
                    </n-tag>
                  </n-space>
                </template>
                <template #description>
                  <n-space>
                    <n-button size="tiny" @click="controlService(svc.name, 'start')" :disabled="svc.status === 'running'">
                      {{ t('nodes.start') }}
                    </n-button>
                    <n-button size="tiny" @click="controlService(svc.name, 'stop')" :disabled="svc.status !== 'running'">
                      {{ t('nodes.stop') }}
                    </n-button>
                    <n-button size="tiny" @click="controlService(svc.name, 'restart')">
                      {{ t('nodes.restart') }}
                    </n-button>
                  </n-space>
                </template>
              </n-thing>
            </n-list-item>
          </n-list>
          <n-empty v-else :description="t('nodes.noServices')" />
        </n-card>
      </n-tab-pane>

      <n-tab-pane name="execute" :tab="t('nodes.execute')">
        <n-card :title="t('nodes.callPlugin')">
          <n-form :model="executeForm" label-placement="top">
            <n-form-item :label="t('nodes.selectPlugin')">
              <n-select
                v-model:value="executeForm.plugin"
                :options="pluginSelectOptions"
                :placeholder="t('nodes.selectPlugin')"
                filterable
              />
            </n-form-item>
            <n-form-item :label="t('nodes.function')">
              <n-input v-model:value="executeForm.function" placeholder="list" />
            </n-form-item>
            <n-form-item :label="t('nodes.arguments')">
              <n-dynamic-input v-model:value="executeForm.args" placeholder="arg1" />
            </n-form-item>
            <n-form-item>
              <n-button type="primary" @click="executePlugin" :loading="executing">
                {{ t('nodes.execute') }}
              </n-button>
            </n-form-item>
          </n-form>

          <n-divider v-if="executeResult" />

          <n-card v-if="executeResult" :bordered="false" size="small">
            <template #header>{{ t('nodes.result') }}</template>
            <n-code :code="executeResult" language="json" word-wrap />
          </n-card>
        </n-card>
      </n-tab-pane>
    </n-tabs>

    <!-- Edit Node Modal -->
    <n-modal v-model:show="showEditModal" preset="card" :title="t('nodes.edit')" style="width: 500px">
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
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditModal = false">{{ t('nodes.cancel') }}</n-button>
          <n-button type="primary" @click="handleUpdate" :loading="saving">{{ t('nodes.save') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Install Plugin Modal -->
    <n-modal v-model:show="showInstallPluginModal" preset="card" :title="t('nodes.installPlugin')" style="width: 500px">
      <n-form :model="installForm" label-placement="top">
        <n-form-item :label="t('nodes.selectFromRepo')">
          <n-select
            v-model:value="installForm.plugin_id"
            :options="repoPluginOptions"
            :placeholder="t('nodes.selectPlugin')"
            filterable
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showInstallPluginModal = false">{{ t('nodes.cancel') }}</n-button>
          <n-button type="primary" @click="handleInstallPlugin" :loading="installing">{{ t('nodes.install') }}</n-button>
        </n-space>
      </template>
    </n-modal>

    <!-- Add Client Modal -->
    <n-modal v-model:show="showAddClientModal" preset="card" :title="t('nodes.addClient')" style="width: 500px">
      <n-form :model="clientForm" :rules="clientRules" ref="clientFormRef" label-placement="top">
        <n-form-item :label="t('nodes.clientCN')" path="client_cn">
          <n-input v-model:value="clientForm.client_cn" placeholder="client001" />
        </n-form-item>
        <n-form-item :label="t('nodes.role')" path="role">
          <n-select
            v-model:value="clientForm.role"
            :options="roleOptions"
          />
        </n-form-item>
        <n-form-item :label="t('nodes.allowedPlugins')">
          <n-select
            v-model:value="clientForm.allowed_plugins"
            :options="pluginNameOptions"
            multiple
            placeholder="*"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="showAddClientModal = false">{{ t('nodes.cancel') }}</n-button>
          <n-button type="primary" @click="handleAddClient" :loading="addingClient">{{ t('nodes.add') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NTabs, NTabPane, NCard, NButton, NSpace, NIcon, NTag, NDescriptions,
  NDescriptionsItem, NDataTable, NEmpty, NList, NListItem, NThing,
  NForm, NFormItem, NInput, NInputNumber, NSelect, NDynamicInput,
  NCode, NDivider, NPopconfirm, useMessage
} from 'naive-ui'
import {
  ArrowBackOutline, FlashOutline, CreateOutline, AddOutline,
  RefreshOutline, TrashOutline
} from '@vicons/ionicons5'
import api from '@/api'

const route = useRoute()
const { t } = useI18n()
const message = useMessage()

const nodeId = computed(() => Number(route.params.id))

const node = ref<any>(null)
const nodePlugins = ref<any[]>([])
const clients = ref<any[]>([])
const services = ref<any[]>([])
const repoPlugins = ref<any[]>([])

const loadingPlugins = ref(false)
const loadingClients = ref(false)
const testing = ref(false)
const saving = ref(false)
const installing = ref(false)
const addingClient = ref(false)
const executing = ref(false)

const showEditModal = ref(false)
const showInstallPluginModal = ref(false)
const showAddClientModal = ref(false)

const editForm = ref({ name: '', host: '', port: 18888 })
const installForm = ref({ plugin_id: null })
const clientForm = ref({ client_cn: '', role: 'viewer', allowed_plugins: [] as string[] })
const clientFormRef = ref()

const executeForm = ref({ plugin: '', function: '', args: [] as string[] })
const executeResult = ref('')

const statusType = computed(() => {
  if (node.value?.status === 'online') return 'success'
  if (node.value?.status === 'offline') return 'error'
  return 'default'
})

const pluginColumns = [
  { title: () => t('plugins.name'), key: 'name', width: 150 },
  { title: () => t('plugins.version'), key: 'version', width: 100 },
  { title: () => t('plugins.type'), key: 'type', width: 100 },
  {
    title: () => t('nodes.status'),
    key: 'enabled',
    width: 100,
    render: (row: any) => h(NTag, { type: row.enabled ? 'success' : 'default', size: 'small' },
      { default: () => row.enabled ? t('nodes.enabled') : t('nodes.disabled') })
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
          onClick: () => togglePlugin(row, !row.enabled)
        }, { default: () => row.enabled ? t('nodes.disable') : t('nodes.enable') }),
        h(NPopconfirm, {
          onPositiveClick: () => uninstallPlugin(row.name)
        }, {
          trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error' },
            { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
          default: () => t('nodes.uninstallConfirm')
        })
      ]
    })
  }
]

const clientColumns = [
  { title: 'CN', key: 'client_cn', width: 200 },
  { title: () => t('nodes.role'), key: 'role', width: 120 },
  { title: () => t('nodes.allowedPlugins'), key: 'allowed_plugins', ellipsis: { tooltip: true } },
  { title: () => t('nodes.created'), key: 'created_at', width: 180,
    render: (row: any) => new Date(row.created_at).toLocaleString() },
  {
    title: () => t('nodes.actions'),
    key: 'actions',
    width: 80,
    render: (row: any) => h(NPopconfirm, {
      onPositiveClick: () => deleteClient(row.id)
    }, {
      trigger: () => h(NButton, { size: 'small', quaternary: true, type: 'error', circle: true },
        { icon: () => h(NIcon, null, { default: () => h(TrashOutline) }) }),
      default: () => t('nodes.deleteConfirm')
    })
  }
]

const roleOptions = [
  { label: 'Admin', value: 'admin' },
  { label: 'Operator', value: 'operator' },
  { label: 'Viewer', value: 'viewer' }
]

const pluginSelectOptions = computed(() =>
  nodePlugins.value.map((p: any) => ({ label: p.name, value: p.name }))
)

const pluginNameOptions = [
  { label: '* (All)', value: '*' },
  { label: 'shell', value: 'shell' },
  { label: 'file_transfer', value: 'file_transfer' },
  { label: 'terminal', value: 'terminal' },
  { label: 'proxy', value: 'proxy' },
  { label: 'plugin_mgr', value: 'plugin_mgr' }
]

const repoPluginOptions = computed(() =>
  repoPlugins.value.map((p: any) => ({ label: `${p.name} - ${p.version}`, value: p.id }))
)

const clientRules = {
  client_cn: { required: true, message: t('nodes.clientCNRequired'), trigger: 'blur' },
  role: { required: true, message: t('nodes.roleRequired'), trigger: 'change' }
}

async function loadNode() {
  try {
    const res = await api.get(`/nodes/${nodeId.value}`)
    node.value = res.data
    editForm.value = {
      name: node.value.name,
      host: node.value.host,
      port: node.value.port
    }
  } catch {
    message.error(t('nodes.failedToLoad'))
  }
}

async function loadNodePlugins() {
  loadingPlugins.value = true
  try {
    const res = await api.get(`/nodes/${nodeId.value}/plugins`)
    nodePlugins.value = res.data || []
  } catch {
    message.error(t('nodes.failedToLoadPlugins'))
  } finally {
    loadingPlugins.value = false
  }
}

async function loadClients() {
  loadingClients.value = true
  try {
    const res = await api.get(`/nodes/${nodeId.value}/clients`)
    clients.value = res.data || []
  } catch {
    message.error(t('nodes.failedToLoadClients'))
  } finally {
    loadingClients.value = false
  }
}

async function loadRepoPlugins() {
  try {
    const res = await api.get('/plugins')
    repoPlugins.value = res.data || []
  } catch {
    console.error('Failed to load repo plugins')
  }
}

async function loadServices() {
  try {
    const res = await api.get(`/nodes/${nodeId.value}/services`)
    services.value = res.data || []
  } catch {
    message.error(t('nodes.failedToLoadServices'))
  }
}

async function testConnection() {
  testing.value = true
  try {
    await api.post(`/nodes/${nodeId.value}/connect`)
    message.success(t('nodes.connectionSuccess'))
    loadNode()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.connectionFailed'))
  } finally {
    testing.value = false
  }
}

async function handleUpdate() {
  saving.value = true
  try {
    await api.put(`/nodes/${nodeId.value}`, editForm.value)
    message.success(t('nodes.nodeUpdated'))
    showEditModal.value = false
    loadNode()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToUpdate'))
  } finally {
    saving.value = false
  }
}

async function handleInstallPlugin() {
  if (!installForm.value.plugin_id) {
    message.warning(t('nodes.selectPlugin'))
    return
  }
  installing.value = true
  try {
    await api.post(`/nodes/${nodeId.value}/plugins/install`, { plugin_id: installForm.value.plugin_id })
    message.success(t('nodes.pluginInstalled'))
    showInstallPluginModal.value = false
    installForm.value.plugin_id = null
    loadNodePlugins()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToInstall'))
  } finally {
    installing.value = false
  }
}

async function togglePlugin(plugin: any, enable: boolean) {
  try {
    const action = enable ? 'enable' : 'disable'
    await api.post(`/nodes/${nodeId.value}/plugins/${plugin.name}/${action}`)
    message.success(enable ? t('nodes.pluginEnabled') : t('nodes.pluginDisabled'))
    loadNodePlugins()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToToggle'))
  }
}

async function uninstallPlugin(name: string) {
  try {
    await api.delete(`/nodes/${nodeId.value}/plugins/${name}`)
    message.success(t('nodes.pluginUninstalled'))
    loadNodePlugins()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToUninstall'))
  }
}

async function handleAddClient() {
  try {
    await clientFormRef.value?.validate()
  } catch {
    return
  }
  addingClient.value = true
  try {
    await api.post(`/nodes/${nodeId.value}/clients`, clientForm.value)
    message.success(t('nodes.clientAdded'))
    showAddClientModal.value = false
    clientForm.value = { client_cn: '', role: 'viewer', allowed_plugins: [] }
    loadClients()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToAddClient'))
  } finally {
    addingClient.value = false
  }
}

async function deleteClient(id: number) {
  try {
    await api.delete(`/nodes/${nodeId.value}/clients/${id}`)
    message.success(t('nodes.clientDeleted'))
    loadClients()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToDeleteClient'))
  }
}

async function controlService(name: string, action: string) {
  try {
    await api.post(`/nodes/${nodeId.value}/services/${name}/${action}`)
    message.success(t('nodes.serviceActionSuccess'))
    loadServices()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('nodes.failedToControlService'))
  }
}

async function executePlugin() {
  if (!executeForm.value.plugin || !executeForm.value.function) {
    message.warning(t('nodes.fillRequiredFields'))
    return
  }
  executing.value = true
  try {
    const res = await api.post(`/nodes/${nodeId.value}/execute`, {
      plugin: executeForm.value.plugin,
      function: executeForm.value.function,
      args: executeForm.value.args
    })
    executeResult.value = JSON.stringify(res.data, null, 2)
  } catch (e: any) {
    executeResult.value = JSON.stringify({ error: e?.response?.data?.error || 'Failed' }, null, 2)
  } finally {
    executing.value = false
  }
}

onMounted(async () => {
  await Promise.all([
    loadNode(),
    loadNodePlugins(),
    loadClients(),
    loadRepoPlugins(),
    loadServices()
  ])
})
</script>

<style scoped>
.node-detail-page {
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