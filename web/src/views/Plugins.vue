<template>
  <div class="plugins-page max-w-6xl">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-semibold">{{ t('plugins.title') }}</h2>
      <Button @click="showUploadModal = true">
        <Upload class="h-4 w-4 mr-2" />
        {{ t('plugins.uploadPlugin') }}
      </Button>
    </div>

    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <Card v-for="plugin in data" :key="plugin.id" class="plugin-card">
        <CardHeader>
          <div class="flex items-center justify-between">
            <span class="font-semibold">{{ plugin.name }}</span>
            <Badge variant="secondary">{{ plugin.plugin_type }}</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <div class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ t('plugins.version') }}</span>
              <span>{{ plugin.version }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ t('plugins.size') }}</span>
              <span>{{ formatSize(plugin.file_size) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ t('plugins.checksum') }}</span>
              <code class="text-xs bg-muted px-1 rounded">{{ plugin.checksum?.substring(0, 16) }}...</code>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">{{ t('plugins.uploaded') }}</span>
              <span>{{ formatDate(plugin.created_at) }}</span>
            </div>
          </div>
        </CardContent>
        <CardFooter class="flex justify-end gap-2">
          <Button variant="ghost" size="sm" @click="handleDownload(plugin.id)">
            <Download class="h-4 w-4" />
          </Button>
          <AlertDialog>
            <AlertDialogTrigger as-child>
              <Button variant="ghost" size="sm">
                <Trash class="h-4 w-4 text-destructive" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{{ t('plugins.deleteConfirm') }}</AlertDialogTitle>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{{ t('plugins.cancel') }}</AlertDialogCancel>
                <AlertDialogAction @click="handleDelete(plugin.id)">{{ t('plugins.delete') }}</AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </CardFooter>
      </Card>
    </div>

    <div v-if="data.length === 0 && !loading" class="text-center py-12 text-muted-foreground">
      {{ t('plugins.noPlugins') }}
    </div>

    <Dialog v-model:open="showUploadModal">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{{ t('plugins.uploadPlugin') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleUpload" class="space-y-4">
          <div class="space-y-2">
            <Label for="plugin-name">{{ t('plugins.pluginName') }}</Label>
            <Input id="plugin-name" v-model="uploadForm.name" placeholder="my-plugin" />
          </div>
          <div class="space-y-2">
            <Label for="plugin-version">{{ t('plugins.version') }}</Label>
            <Input id="plugin-version" v-model="uploadForm.version" placeholder="1.0.0" />
          </div>
          <div class="space-y-2">
            <Label for="plugin-type">{{ t('plugins.pluginType') }}</Label>
            <Select v-model="uploadForm.type">
              <SelectTrigger>
                <SelectValue :placeholder="t('plugins.selectType')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="service">{{ t('plugins.types.service') }}</SelectItem>
                <SelectItem value="command">{{ t('plugins.types.command') }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="plugin-desc">{{ t('plugins.description') }}</Label>
            <Textarea id="plugin-desc" v-model="uploadForm.description" placeholder="Plugin description..." />
          </div>
          <div class="space-y-2">
            <Label>{{ t('plugins.selectFile') }}</Label>
            <Input type="file" accept=".wasm" @change="handleFileChange" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="closeUploadModal">{{ t('plugins.cancel') }}</Button>
            <Button type="submit" :disabled="uploading">{{ t('plugins.upload') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Upload, Download, Trash } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import {Card, CardHeader, CardTitle, CardContent, CardFooter} from '@/components/ui'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Textarea from '@/components/ui/Textarea.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui'
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from '@/components/ui'
import { useToast } from '@/components/ui/useToast.ts'
import api from '@/api'

const { t } = useI18n()
const { toast } = useToast()

const loading = ref(true)
const uploading = ref(false)
const data = ref<any[]>([])
const showUploadModal = ref(false)

const uploadForm = ref({
  name: '',
  version: '',
  type: '',
  description: ''
})
const selectedFile = ref<File | null>(null)

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
    toast({ title: 'Error', description: t('plugins.failedToLoad'), variant: 'destructive' })
  } finally {
    loading.value = false
  }
}

function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files[0]) {
    selectedFile.value = target.files[0]
  }
}

async function handleUpload() {
  if (!uploadForm.value.name || !uploadForm.value.version || !uploadForm.value.type) return
  if (!selectedFile.value) {
    toast({ title: 'Error', description: t('plugins.fileRequired'), variant: 'destructive' })
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
    toast({ title: t('plugins.pluginUploaded') })
    closeUploadModal()
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('plugins.uploadFailed'), variant: 'destructive' })
  } finally {
    uploading.value = false
  }
}

function closeUploadModal() {
  showUploadModal.value = false
  uploadForm.value = { name: '', version: '', type: '', description: '' }
  selectedFile.value = null
}

function handleDownload(id: number) {
  window.open(`/api/plugins/${id}/download`, '_blank')
}

async function handleDelete(id: number) {
  try {
    await api.delete(`/plugins/${id}`)
    toast({ title: t('plugins.pluginDeleted') })
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('plugins.deleteFailed'), variant: 'destructive' })
  }
}

onMounted(loadData)
</script>