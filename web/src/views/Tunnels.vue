<template>
  <div class="tunnels-page max-w-7xl">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-semibold">{{ t('tunnels.title') }}</h2>
      <Button @click="showAddModal = true">
        <Plus class="h-4 w-4 mr-2" />
        {{ t('tunnels.addTunnel') }}
      </Button>
    </div>

    <Card>
      <CardHeader>
        <div class="flex items-center justify-between">
          <span>{{ t('tunnels.total', { count: data.length }) }}</span>
          <Button variant="outline" size="sm" @click="loadData">
            <Refresh class="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('tunnels.name') }}</TableHead>
              <TableHead>{{ t('tunnels.type') }}</TableHead>
              <TableHead>{{ t('tunnels.status') }}</TableHead>
              <TableHead>{{ t('tunnels.transportMode') }}</TableHead>
              <TableHead>{{ t('tunnels.localAddr') }}</TableHead>
              <TableHead>{{ t('tunnels.remoteAddr') }}</TableHead>
              <TableHead>{{ t('tunnels.e2eEnabled') }}</TableHead>
              <TableHead>{{ t('tunnels.bytesIn') }}</TableHead>
              <TableHead>{{ t('tunnels.bytesOut') }}</TableHead>
              <TableHead>{{ t('tunnels.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in data" :key="row.id">
              <TableCell>{{ row.name }}</TableCell>
              <TableCell>{{ row.type }}</TableCell>
              <TableCell>
                <Badge :variant="getStatusVariant(row.status)">{{ row.status }}</Badge>
              </TableCell>
              <TableCell>{{ row.transport_mode }}</TableCell>
              <TableCell>{{ row.local_addr }}</TableCell>
              <TableCell>{{ row.remote_addr }}</TableCell>
              <TableCell>
                <Badge :variant="row.e2e_enabled ? 'default' : 'secondary'">{{ row.e2e_enabled ? 'Yes' : 'No' }}</Badge>
              </TableCell>
              <TableCell>{{ formatBytes(row.bytes_in || 0) }}</TableCell>
              <TableCell>{{ formatBytes(row.bytes_out || 0) }}</TableCell>
              <TableCell>
                <div class="flex gap-2">
                  <Button variant="ghost" size="icon" @click="handleToggle(row.id)" :title="row.status === 'active' ? t('tunnels.pause') : t('tunnels.resume')">
                    <Pause v-if="row.status === 'active'" class="h-4 w-4" />
                    <Play v-else class="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="icon" @click="openEditModal(row)">
                    <Pencil class="h-4 w-4" />
                  </Button>
                  <AlertDialog>
                    <AlertDialogTrigger as-child>
                      <Button variant="ghost" size="icon">
                        <Trash class="h-4 w-4 text-destructive" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>{{ t('tunnels.deleteConfirm') }}</AlertDialogTitle>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>{{ t('tunnels.cancel') }}</AlertDialogCancel>
                        <AlertDialogAction @click="handleDelete(row.id)">{{ t('tunnels.delete') }}</AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <Dialog v-model:open="showAddModal">
      <DialogContent class="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{{ t('tunnels.addTunnel') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleAdd" class="space-y-4">
          <div class="space-y-2">
            <Label for="name">{{ t('tunnels.name') }}</Label>
            <Input id="name" v-model="formValue.name" placeholder="My Tunnel" />
          </div>
          <div class="space-y-2">
            <Label for="type">{{ t('tunnels.type') }}</Label>
            <Select v-model="formValue.type">
              <SelectTrigger>
                <SelectValue :placeholder="t('tunnels.selectType')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="port_forward">Port Forward</SelectItem>
                <SelectItem value="socks5">SOCKS5</SelectItem>
                <SelectItem value="http">HTTP Proxy</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="node">{{ t('tunnels.node') }}</Label>
            <Select v-model="formValue.node_id">
              <SelectTrigger>
                <SelectValue :placeholder="t('tunnels.selectNode')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="n in nodeOptions" :key="n.value" :value="n.value">{{ n.label }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="transport">{{ t('tunnels.transportMode') }}</Label>
            <Select v-model="formValue.transport_mode">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="auto">Auto</SelectItem>
                <SelectItem value="direct">Direct</SelectItem>
                <SelectItem value="relay">Relay</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="local">{{ t('tunnels.localAddr') }}</Label>
            <Input id="local" v-model="formValue.local_addr" placeholder=":8080" />
          </div>
          <div class="space-y-2">
            <Label for="remote">{{ t('tunnels.remoteAddr') }}</Label>
            <Input id="remote" v-model="formValue.remote_addr" placeholder="10.0.0.1:80" />
          </div>
          <div class="space-y-2">
            <Label for="obfuscation">{{ t('tunnels.obfuscationMode') }}</Label>
            <Select v-model="formValue.obfuscation_mode">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">None</SelectItem>
                <SelectItem value="http2_masquerade">HTTP/2 Masquerade</SelectItem>
                <SelectItem value="domain_fronting">Domain Fronting</SelectItem>
                <SelectItem value="traffic_padding">Traffic Padding</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="flex items-center gap-2">
            <Label for="e2e">{{ t('tunnels.e2eEnabled') }}</Label>
            <input id="e2e" type="checkbox" v-model="formValue.e2e_enabled" class="-checkbox" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="showAddModal = false">{{ t('tunnels.cancel') }}</Button>
            <Button type="submit" :disabled="saving">{{ t('tunnels.add') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showEditModal">
      <DialogContent class="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{{ t('tunnels.editTunnel') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleEdit" class="space-y-4">
          <div class="space-y-2">
            <Label for="edit-name">{{ t('tunnels.name') }}</Label>
            <Input id="edit-name" v-model="editFormValue.name" />
          </div>
          <div class="space-y-2">
            <Label for="edit-type">{{ t('tunnels.type') }}</Label>
            <Select v-model="editFormValue.type">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="port_forward">Port Forward</SelectItem>
                <SelectItem value="socks5">SOCKS5</SelectItem>
                <SelectItem value="http">HTTP Proxy</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="edit-transport">{{ t('tunnels.transportMode') }}</Label>
            <Select v-model="editFormValue.transport_mode">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="auto">Auto</SelectItem>
                <SelectItem value="direct">Direct</SelectItem>
                <SelectItem value="relay">Relay</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="edit-local">{{ t('tunnels.localAddr') }}</Label>
            <Input id="edit-local" v-model="editFormValue.local_addr" />
          </div>
          <div class="space-y-2">
            <Label for="edit-remote">{{ t('tunnels.remoteAddr') }}</Label>
            <Input id="edit-remote" v-model="editFormValue.remote_addr" />
          </div>
          <div class="space-y-2">
            <Label for="edit-obfuscation">{{ t('tunnels.obfuscationMode') }}</Label>
            <Select v-model="editFormValue.obfuscation_mode">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">None</SelectItem>
                <SelectItem value="http2_masquerade">HTTP/2 Masquerade</SelectItem>
                <SelectItem value="domain_fronting">Domain Fronting</SelectItem>
                <SelectItem value="traffic_padding">Traffic Padding</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="edit-status">{{ t('tunnels.status') }}</Label>
            <Select v-model="editFormValue.status">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="paused">Paused</SelectItem>
                <SelectItem value="stopped">Stopped</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="showEditModal = false">{{ t('tunnels.cancel') }}</Button>
            <Button type="submit" :disabled="saving">{{ t('tunnels.save') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Refresh, Pencil, Trash, Pause, Play } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import {Card, CardHeader, CardTitle, CardContent} from '@/components/ui'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui'
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from '@/components/ui'
import { useToast } from '@/components/ui/useToast.ts'
import api from '@/api'
import { tunnelApi, type Tunnel, type CreateTunnelRequest, type UpdateTunnelRequest } from '@/api/tunnel'

const { t } = useI18n()
const { toast } = useToast()

const loading = ref(true)
const saving = ref(false)
const data = ref<Tunnel[]>([])
const nodes = ref<any[]>([])
const showAddModal = ref(false)
const showEditModal = ref(false)
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

const nodeOptions = ref<{ label: string; value: string }[]>([])

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function getStatusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  const variants: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
    active: 'default',
    paused: 'secondary',
    stopped: 'outline'
  }
  return variants[status] || 'outline'
}

async function loadData() {
  loading.value = true
  try {
    data.value = await tunnelApi.list()
  } catch {
    toast({ title: 'Error', description: t('tunnels.failedToLoad'), variant: 'destructive' })
  } finally {
    loading.value = false
  }
}

async function loadNodes() {
  try {
    const res = await api.get('/nodes')
    nodes.value = res.data || []
    nodeOptions.value = nodes.value.map((n: any) => ({
      label: n.name,
      value: String(n.id)
    }))
  } catch (e) {
    console.error('Failed to load nodes', e)
  }
}

async function handleAdd() {
  saving.value = true
  try {
    await tunnelApi.create(formValue.value)
    toast({ title: t('tunnels.tunnelCreated') })
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
    toast({ title: 'Error', description: e?.response?.data?.error || t('tunnels.failedToCreate'), variant: 'destructive' })
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
    toast({ title: t('tunnels.tunnelUpdated') })
    showEditModal.value = false
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('tunnels.failedToUpdate'), variant: 'destructive' })
  } finally {
    saving.value = false
  }
}

async function handleToggle(id: number) {
  try {
    const res = await tunnelApi.toggle(id)
    toast({ title: res.status === 'active' ? t('tunnels.tunnelResumed') : t('tunnels.tunnelPaused') })
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('tunnels.failedToToggle'), variant: 'destructive' })
  }
}

async function handleDelete(id: number) {
  try {
    await tunnelApi.delete(id)
    toast({ title: t('tunnels.tunnelDeleted') })
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('tunnels.failedToDelete'), variant: 'destructive' })
  }
}

onMounted(() => {
  loadData()
  loadNodes()
})
</script>

<style scoped>
.checkbox {
  width: 16px;
  height: 16px;
}
</style>