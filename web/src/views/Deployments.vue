<template>
  <div class="deployments-page max-w-6xl">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-semibold">{{ t('deployments.title') }}</h2>
      <Button @click="showDeployModal = true">
        <Upload class="h-4 w-4 mr-2" />
        {{ t('deployments.deployPlugin') }}
      </Button>
    </div>

    <Card>
      <CardHeader>
        <div class="flex items-center gap-2">
          <Badge variant="outline">{{ t('deployments.total') }}: {{ data.length }}</Badge>
          <Badge variant="default">{{ t('deployments.deployed') }}: {{ deployedCount }}</Badge>
          <Badge variant="secondary">{{ t('deployments.pending') }}: {{ pendingCount }}</Badge>
          <Button variant="ghost" size="icon" @click="loadData">
            <Refresh class="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>{{ t('deployments.plugin') }}</TableHead>
              <TableHead>{{ t('deployments.node') }}</TableHead>
              <TableHead>{{ t('deployments.version') }}</TableHead>
              <TableHead>{{ t('deployments.status') }}</TableHead>
              <TableHead>{{ t('deployments.deployedBy') }}</TableHead>
              <TableHead>{{ t('deployments.created') }}</TableHead>
              <TableHead>{{ t('deployments.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in data" :key="row.id">
              <TableCell>{{ row.id }}</TableCell>
              <TableCell>{{ row.plugin_name }}</TableCell>
              <TableCell>{{ row.node_name }}</TableCell>
              <TableCell>{{ row.version }}</TableCell>
              <TableCell>
                <Badge :variant="getStatusVariant(row.status)">{{ row.status }}</Badge>
              </TableCell>
              <TableCell>{{ row.deployed_by }}</TableCell>
              <TableCell>{{ new Date(row.created_at).toLocaleString() }}</TableCell>
              <TableCell>
                <AlertDialog v-if="row.status === 'pending'">
                  <AlertDialogTrigger as-child>
                    <Button variant="ghost" size="sm">{{ t('deployments.cancel') }}</Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>{{ t('deployments.cancelConfirm') }}</AlertDialogTitle>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>{{ t('common.cancel') }}</AlertDialogCancel>
                      <AlertDialogAction @click="handleCancel(row.id)">{{ t('deployments.cancel') }}</AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <Dialog v-model:open="showDeployModal">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{{ t('deployments.deployPlugin') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleDeploy" class="space-y-4">
          <div class="space-y-2">
            <Label for="plugin">{{ t('deployments.plugin') }}</Label>
            <Select v-model="deployForm.plugin_id">
              <SelectTrigger>
                <SelectValue :placeholder="t('deployments.selectPlugin')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="p in plugins" :key="p.id" :value="p.id">{{ p.name }} - {{ p.version }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="nodes">{{ t('deployments.node') }}</Label>
            <Select v-model="deployForm.node_ids" multiple>
              <SelectTrigger>
                <SelectValue :placeholder="t('deployments.selectNodes')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="n in nodes" :key="n.id" :value="n.id">{{ n.name }} ({{ n.host }})</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="showDeployModal = false">{{ t('common.cancel') }}</Button>
            <Button type="submit" :disabled="saving">{{ t('deployments.deployPlugin') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Upload, Refresh } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import {Card, CardHeader, CardTitle, CardContent} from '@/components/ui'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui'
import Badge from '@/components/ui/Badge.vue'
import Label from '@/components/ui/Label.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui'
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from '@/components/ui'
import { useToast } from '@/components/ui/useToast.ts'
import api from '@/api'

const { t } = useI18n()
const { toast } = useToast()

const loading = ref(true)
const saving = ref(false)
const data = ref<any[]>([])
const nodes = ref<any[]>([])
const plugins = ref<any[]>([])
const showDeployModal = ref(false)

const deployForm = ref<{ plugin_id: number | null; node_ids: number[] }>({ plugin_id: null, node_ids: [] })

const deployedCount = computed(() => data.value.filter(d => d.status === 'deployed').length)
const pendingCount = computed(() => data.value.filter(d => d.status === 'pending' || d.status === 'in_progress').length)

function getStatusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  const variants: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
    deployed: 'default',
    pending: 'secondary',
    in_progress: 'secondary',
    failed: 'destructive',
    cancelled: 'outline'
  }
  return variants[status] || 'outline'
}

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/deployments')
    data.value = res.data || []
  } catch {
    toast({ title: 'Error', description: t('deployments.failedToLoad'), variant: 'destructive' })
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
    toast({ title: 'Error', description: t('deployments.failedToLoad'), variant: 'destructive' })
  }
}

async function handleDeploy() {
  if (!deployForm.value.plugin_id || deployForm.value.node_ids.length === 0) {
    toast({ title: 'Error', description: t('deployments.pleaseSelectNodes'), variant: 'destructive' })
    return
  }

  saving.value = true
  try {
    await api.post('/deployments', {
      plugin_id: deployForm.value.plugin_id,
      node_ids: deployForm.value.node_ids
    })
    toast({ title: t('deployments.deploymentInitiated') })
    showDeployModal.value = false
    deployForm.value = { plugin_id: null, node_ids: [] }
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('deployments.failedToDeploy'), variant: 'destructive' })
  } finally {
    saving.value = false
  }
}

async function handleCancel(id: number) {
  try {
    await api.post(`/deployments/${id}/cancel`)
    toast({ title: t('deployments.deploymentCancelled') })
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('deployments.failedToCancel'), variant: 'destructive' })
  }
}

onMounted(async () => {
  await Promise.all([loadData(), loadNodesAndPlugins()])
})
</script>