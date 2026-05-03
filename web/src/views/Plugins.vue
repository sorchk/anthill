<template>
  <div class="plugins-page">
    <div class="page-header">
      <h2>{{ t('plugins.title') }}</h2>
      <n-button type="primary" @click="showUploadModal = true">
        <template #icon>
          <n-icon><cloud-upload-outline /></n-icon>
        </template>
        {{ t('plugins.uploadPlugin') }}
      </n-button>
    </div>

    <n-grid :cols="3" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
      <n-gi v-for="plugin in data" :key="plugin.id" span="3 m:1">
        <n-card class="plugin-card" bordered hoverable>
          <template #header>
            <n-space justify="space-between" align="center">
              <span class="plugin-name">{{ plugin.name }}</span>
              <n-tag type="info" size="small">{{ plugin.plugin_type }}</n-tag>
            </n-space>
          </template>

          <div class="plugin-info">
            <n-descriptions :column="1" size="small">
              <n-descriptions-item :label="t('plugins.version')">{{ plugin.version }}</n-descriptions-item>
              <n-descriptions-item :label="t('plugins.size')">{{ formatSize(plugin.file_size) }}</n-descriptions-item>
              <n-descriptions-item :label="t('plugins.checksum')">
                <n-tooltip trigger="hover">
                  <template #trigger>
                    <code class="checksum">{{ plugin.checksum?.substring(0, 16) }}...</code>
                  </template>
                  {{ plugin.checksum }}
                </n-tooltip>
              </n-descriptions-item>
              <n-descriptions-item :label="t('plugins.uploaded')">
                {{ formatDate(plugin.created_at) }}
              </n-descriptions-item>
            </n-descriptions>
          </div>

          <template #footer>
            <n-space justify="end">
              <n-button size="small" quaternary @click="handleDownload(plugin.id)">
                <template #icon>
                  <n-icon><download-outline /></n-icon>
                </template>
              </n-button>
              <n-popconfirm @positive-click="handleDelete(plugin.id)">
                <template #trigger>
                  <n-button size="small" quaternary type="error">
                    <template #icon>
                      <n-icon><trash-outline /></n-icon>
                    </template>
                  </n-button>
                </template>
                {{ t('plugins.deleteConfirm') }}
              </n-popconfirm>
            </n-space>
          </template>
        </n-card>
      </n-gi>
    </n-grid>

    <n-empty v-if="data.length === 0 && !loading" :description="t('plugins.noPlugins')" style="margin-top: 40px" />

    <!-- Upload Modal -->
    <n-modal v-model:show="showUploadModal" preset="card" :title="t('plugins.uploadPlugin')" style="width: 500px">
      <n-form :model="uploadForm" :rules="uploadRules" ref="uploadFormRef" label-placement="top">
        <n-form-item :label="t('plugins.pluginName')" path="name">
          <n-input v-model:value="uploadForm.name" placeholder="my-plugin" />
        </n-form-item>
        <n-form-item :label="t('plugins.version')" path="version">
          <n-input v-model:value="uploadForm.version" placeholder="1.0.0" />
        </n-form-item>
        <n-form-item :label="t('plugins.pluginType')" path="type">
          <n-select
            v-model:value="uploadForm.type"
            :options="typeOptions"
            :placeholder="t('plugins.selectType')"
          />
        </n-form-item>
        <n-form-item :label="t('plugins.description')" path="description">
          <n-input v-model:value="uploadForm.description" type="textarea" placeholder="Plugin description..." />
        </n-form-item>
        <n-form-item :label="t('plugins.selectFile')" path="file">
          <n-upload
            ref="uploadRef"
            :max-size="10 * 1024 * 1024"
            :file-list="fileList"
            @change="handleFileChange"
            @remove="handleFileRemove"
            @before-upload="beforeUpload"
            :custom-request="customRequest"
          >
            <n-button>{{ t('plugins.selectFile') }}</n-button>
          </n-upload>
        </n-form-item>
      </n-form>
      <template #footer>
        <n-space justify="end">
          <n-button @click="closeUploadModal">{{ t('plugins.cancel') }}</n-button>
          <n-button type="primary" @click="handleUpload" :loading="uploading">{{ t('plugins.upload') }}</n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NGrid, NGi, NCard, NButton, NSpace, NIcon, NTag, NEmpty, NModal, NForm,
  NFormItem, NInput, NSelect, NDescriptions, NDescriptionsItem, NTooltip,
  NPopconfirm, NUpload, useMessage, useNotification
} from 'naive-ui'
import {
  CloudUploadOutline, DownloadOutline, TrashOutline
} from '@vicons/ionicons5'
import api from '@/api'

const { t } = useI18n()
const message = useMessage()
const notification = useNotification()

const loading = ref(true)
const uploading = ref(false)
const data = ref<any[]>([])
const showUploadModal = ref(false)
const uploadFormRef = ref()
const uploadRef = ref()

const uploadForm = ref({
  name: '',
  version: '',
  type: '',
  description: ''
})
const fileList = ref<any[]>([])
const selectedFile = ref<File | null>(null)

const typeOptions = [
  { label: () => t('plugins.types.service'), value: 'service' },
  { label: () => t('plugins.types.command'), value: 'command' }
]

const uploadRules = {
  name: { required: true, message: t('plugins.pluginName') + ' is required', trigger: 'blur' },
  version: { required: true, message: t('plugins.version') + ' is required', trigger: 'blur' },
  type: { required: true, message: t('plugins.pluginType') + ' is required', trigger: 'change' },
  file: { required: true, message: t('plugins.fileRequired'), trigger: 'change' }
}

function formatSize(bytes: number) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function formatDate(date: string) {
  return new Date(date).toLocaleDateString()
}

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/plugins')
    data.value = res.data || []
  } catch {
    message.error(t('plugins.failedToLoad'))
  } finally {
    loading.value = false
  }
}

function handleFileChange({ file }: any) {
  selectedFile.value = file.file
}

function handleFileRemove() {
  selectedFile.value = null
}

function beforeUpload({ file }: any) {
  if (!file.name.endsWith('.wasm')) {
    message.error(t('plugins.wasmOnly'))
    return false
  }
  return true
}

async function customRequest({ file, onFinish, onError }: any) {
  const formData = new FormData()
  formData.append('file', file.file)
  formData.append('name', uploadForm.value.name)
  formData.append('version', uploadForm.value.version)
  formData.append('type', uploadForm.value.type)
  formData.append('description', uploadForm.value.description)

  try {
    await api.post('/plugins', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    notification.success({ content: t('plugins.pluginUploaded'), duration: 3000 })
    onFinish()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('plugins.uploadFailed'))
    onError()
  }
}

async function handleUpload() {
  try {
    await uploadFormRef.value?.validate()
  } catch {
    return
  }

  if (!selectedFile.value) {
    message.warning(t('plugins.fileRequired'))
    return
  }

  uploading.value = true

  const formData = new FormData()
  formData.append('file', selectedFile.value)
  formData.append('name', uploadForm.value.name)
  formData.append('version', uploadForm.value.version)
  formData.append('type', uploadForm.value.type)
  formData.append('description', uploadForm.value.description)

  try {
    await api.post('/plugins', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    message.success(t('plugins.pluginUploaded'))
    closeUploadModal()
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('plugins.uploadFailed'))
  } finally {
    uploading.value = false
  }
}

function closeUploadModal() {
  showUploadModal.value = false
  uploadForm.value = { name: '', version: '', type: '', description: '' }
  fileList.value = []
  selectedFile.value = null
}

function handleDownload(id: number) {
  window.open(`/api/plugins/${id}/download`, '_blank')
}

async function handleDelete(id: number) {
  try {
    await api.delete(`/plugins/${id}`)
    message.success(t('plugins.pluginDeleted'))
    loadData()
  } catch (e: any) {
    message.error(e?.response?.data?.error || t('plugins.deleteFailed'))
  }
}

onMounted(loadData)
</script>

<style scoped>
.plugins-page {
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

.plugin-card {
  height: 100%;
}

.plugin-card .plugin-name {
  font-weight: 600;
  font-size: 16px;
}

.plugin-info {
  min-height: 100px;
}

.checksum {
  font-size: 11px;
  background: #f5f5f5;
  padding: 2px 4px;
  border-radius: 4px;
  cursor: pointer;
}
</style>