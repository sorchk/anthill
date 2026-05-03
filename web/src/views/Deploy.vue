<template>
  <div class="deploy-page">
    <div class="page-header">
      <h2>{{ t('deploy.title') }}</h2>
      <n-button type="primary" @click="showDeployDialog = true">
        <template #icon><n-icon><CloudUploadOutline /></n-icon></template>
        {{ t('deploy.newDeployment') }}
      </n-button>
    </div>

    <n-card>
      <n-tabs type="line" animated>
        <n-tab-pane name="tasks" :tab="t('deploy.deployTasks')">
          <n-data-table
            :columns="taskColumns"
            :data="tasks"
            :loading="loading"
            :pagination="pagination"
          />
        </n-tab-pane>
        <n-tab-pane name="templates" :tab="t('deploy.nodeTemplates')">
          <n-empty v-if="templates.length === 0" :description="t('deploy.noTemplates')">
            <template #extra>
              <n-button size="small" @click="showTemplateDialog = true">{{ t('deploy.createTemplate') }}</n-button>
            </template>
          </n-empty>
          <div v-else class="templates-grid">
            <n-card v-for="tmpl in templates" :key="tmpl.id" class="template-card">
              <template #header>{{ tmpl.name }}</template>
              <p>{{ tmpl.description }}</p>
              <p class="template-info">
                <n-text depth="3">{{ tmpl.ssh_user }}@{{ tmpl.ssh_host }}:{{ tmpl.ssh_port }}</n-text>
              </p>
              <template #footer>
                <n-space>
                  <n-button size="small" @click="deployFromTemplate(tmpl)">{{ t('deploy.deployFromTemplate') }}</n-button>
                  <n-button size="small" type="error" ghost @click="deleteTemplate(tmpl.id)">{{ t('deploy.delete') }}</n-button>
                </n-space>
              </template>
            </n-card>
          </div>
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <!-- Deploy Dialog -->
    <n-modal v-model:show="showDeployDialog" preset="card" :title="t('deploy.newDeployment')" style="width: 600px">
      <n-form :model="deployForm" label-placement="top">
        <n-grid :cols="2" :x-gap="12">
          <n-gi>
            <n-form-item :label="t('deploy.sshHost')">
              <n-input v-model:value="deployForm.ssh_host" placeholder="192.168.1.100" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('deploy.sshPort')">
              <n-input-number v-model:value="deployForm.ssh_port" :min="1" :max="65535" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('deploy.sshUser')">
              <n-input v-model:value="deployForm.ssh_user" placeholder="root" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('deploy.authMethod')">
              <n-select
                v-model:value="deployForm.auth_method"
                :options="authOptions"
              />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-form-item v-if="deployForm.auth_method === 'password'" :label="t('deploy.password')">
          <n-input v-model:value="deployForm.ssh_password" type="password" show-password-on="click" />
        </n-form-item>

        <n-form-item v-else :label="t('deploy.sshKey')">
          <n-input v-model:value="deployForm.ssh_key" type="textarea" :rows="4" placeholder="-----BEGIN RSA PRIVATE KEY-----..." />
        </n-form-item>

        <n-form-item :label="t('deploy.runtimeVersion')">
          <n-select v-model:value="deployForm.node_version" :options="versionOptions" />
        </n-form-item>

        <n-form-item :label="t('deploy.preinstallPlugins')">
          <n-checkbox-group v-model:value="deployForm.preinstall_plugins">
            <n-space>
              <n-checkbox v-for="p in availablePlugins" :key="p" :value="p" :label="p" />
            </n-space>
          </n-checkbox-group>
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showDeployDialog = false">{{ t('deploy.cancel') }}</n-button>
          <n-button type="primary" :loading="deploying" @click="startDeploy">{{ t('deploy.deploy') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, h, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NDataTable, NCard, NButton, NIcon, NModal, NForm, NFormItem,
  NInput, NInputNumber, NSelect, NSpace, NGrid, NGi, NTabs, NTabPane,
  NCheckbox, NCheckboxGroup, NEmpty, NText, useMessage
} from 'naive-ui'
import { CloudUploadOutline } from '@vicons/ionicons5'
import api from '@/api'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const tasks = ref<any[]>([])
const templates = ref<any[]>([])
const showDeployDialog = ref(false)
const showTemplateDialog = ref(false)
const deploying = ref(false)

const pagination = { pageSize: 10 }

const deployForm = ref({
  ssh_host: '',
  ssh_port: 22,
  ssh_user: 'root',
  ssh_password: '',
  ssh_key: '',
  auth_method: 'password',
  node_version: 'v1.0.0',
  preinstall_plugins: [] as string[]
})

const authOptions = computed(() => [
  { label: t('deploy.password'), value: 'password' },
  { label: t('deploy.sshKey'), value: 'key' }
])

const versionOptions = [
  { label: 'v1.0.0', value: 'v1.0.0' },
  { label: 'v1.1.0', value: 'v1.1.0' },
  { label: 'latest', value: 'latest' }
]

const availablePlugins = ['shell', 'file_transfer', 'terminal', 'proxy', 'plugin_mgr']

const taskColumns = computed(() => [
  { title: 'ID', key: 'id', width: 80 },
  { title: t('deploy.sshHost'), key: 'ssh_host', ellipsis: { tooltip: true } },
  { title: t('deploy.sshUser'), key: 'ssh_user', width: 100 },
  { title: t('audit.status'), key: 'status', width: 120,
    render: (row: any) => {
      const typeMap: Record<string, string> = {
        pending: 'default',
        in_progress: 'info',
        completed: 'success',
        failed: 'error'
      }
      return h(NText, { type: typeMap[row.status] || 'default' }, { default: () => row.status })
    }
  },
  { title: t('audit.time'), key: 'created_at', width: 180,
    render: (row: any) => new Date(row.created_at).toLocaleString()
  },
  {
    title: 'Log',
    key: 'log',
    width: 80,
    render: (row: any) => h(NButton, { size: 'small', quaternary: true, onClick: () => showLog(row) }, { default: () => t('deploy.viewLog') })
  }
])

async function loadTasks() {
  loading.value = true
  try {
    const res = await api.get('/deploy')
    tasks.value = res.data || []
  } catch {
    message.error(t('deploy.failedToLoad'))
  } finally {
    loading.value = false
  }
}

async function startDeploy() {
  deploying.value = true
  try {
    const payload: any = {
      ssh_host: deployForm.value.ssh_host,
      ssh_port: deployForm.value.ssh_port,
      ssh_user: deployForm.value.ssh_user,
      node_version: deployForm.value.node_version,
      preinstall_plugins: deployForm.value.preinstall_plugins
    }

    if (deployForm.value.auth_method === 'password') {
      payload.ssh_password = deployForm.value.ssh_password
    } else {
      payload.ssh_key = deployForm.value.ssh_key
    }

    await api.post('/deploy', payload)
    message.success(t('deploy.deploymentStarted'))
    showDeployDialog.value = false
    loadTasks()
  } catch (e: any) {
    message.error(e.response?.data?.error || t('deploy.deployFailed'))
  } finally {
    deploying.value = false
  }
}

function showLog(task: any) {
  if (task.log) {
    message.info(task.log.replace(/\n/g, ' | '))
  }
}

function deployFromTemplate(tmpl: any) {
  deployForm.value.ssh_host = tmpl.ssh_host
  deployForm.value.ssh_port = tmpl.ssh_port
  deployForm.value.ssh_user = tmpl.ssh_user
  showDeployDialog.value = true
}

function deleteTemplate(_id: number) {
  message.info(t('deploy.templateNotImplemented'))
}

onMounted(loadTasks)
</script>

<style scoped>
.deploy-page {
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

.templates-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.template-card {
  margin-bottom: 0;
}

.template-info {
  font-size: 12px;
  margin-top: 8px;
}
</style>