<template>
  <div class="nodes-page max-w-6xl">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-semibold">{{ t('nodes.title') }}</h2>
      <Button @click="showAddModal = true">
        <Plus class="h-4 w-4 mr-2" />
        {{ t('nodes.addNode') }}
      </Button>
    </div>

    <Card>
      <CardHeader>
        <div class="flex items-center justify-between">
          <span>{{ t('nodes.total', { count: data.length }) }}</span>
          <Button variant="outline" size="sm" @click="loadData">
            <Refresh class="h-4 w-4 mr-2" />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('nodes.name') }}</TableHead>
              <TableHead>{{ t('nodes.host') }}</TableHead>
              <TableHead>{{ t('nodes.port') }}</TableHead>
              <TableHead>{{ t('nodes.connectMode') }}</TableHead>
              <TableHead>{{ t('nodes.status') }}</TableHead>
              <TableHead>{{ t('nodes.lastSeen') }}</TableHead>
              <TableHead>{{ t('nodes.actions') }}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in data" :key="row.id">
              <TableCell>{{ row.name }}</TableCell>
              <TableCell>{{ row.host }}</TableCell>
              <TableCell>{{ row.port }}</TableCell>
              <TableCell>
                <Badge :variant="row.connect_mode?.startsWith('active') ? 'default' : 'secondary'">
                  {{ row.connect_mode || 'passive_tls' }}
                </Badge>
              </TableCell>
              <TableCell>
                <Badge :variant="row.status === 'online' ? 'default' : row.status === 'offline' ? 'destructive' : 'secondary'">
                  {{ row.status || 'unknown' }}
                </Badge>
              </TableCell>
              <TableCell>{{ row.last_seen ? new Date(row.last_seen).toLocaleString() : '-' }}</TableCell>
              <TableCell>
                <div class="flex gap-2">
                  <Button variant="ghost" size="sm" @click="handleEdit(row)">
                    {{ t('nodes.edit') }}
                  </Button>
                  <Button variant="ghost" size="sm" @click="handleConnect(row.id)">
                    <Zap class="h-4 w-4" />
                  </Button>
                  <AlertDialog>
                    <AlertDialogTrigger as-child>
                      <Button variant="ghost" size="sm">
                        <Trash class="h-4 w-4 text-destructive" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>{{ t('nodes.deleteConfirm') }}</AlertDialogTitle>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>{{ t('nodes.cancel') }}</AlertDialogCancel>
                        <AlertDialogAction @click="handleDelete(row.id)">{{ t('nodes.delete') }}</AlertDialogAction>
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
          <DialogTitle>{{ t('nodes.addNode') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleAdd" class="space-y-4">
          <div class="space-y-2">
            <Label for="name">{{ t('nodes.name') }}</Label>
            <Input id="name" v-model="formValue.name" placeholder="My Node" />
          </div>
          <div class="space-y-2">
            <Label for="host">{{ t('nodes.host') }}</Label>
            <Input id="host" v-model="formValue.host" placeholder="192.168.1.100" />
          </div>
          <div class="space-y-2">
            <Label for="port">{{ t('nodes.port') }}</Label>
            <Input id="port" v-model.number="formValue.port" type="number" :min="1" :max="65535" />
          </div>
          <div class="space-y-2">
            <Label for="connect_mode">{{ t('nodes.connectMode') }}</Label>
            <Select v-model="formValue.connect_mode">
              <SelectTrigger>
                <SelectValue :placeholder="t('nodes.selectConnectMode')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active_tls">Active TLS</SelectItem>
                <SelectItem value="active_wss">Active WSS</SelectItem>
                <SelectItem value="passive_tls">Passive TLS</SelectItem>
                <SelectItem value="passive_wss">Passive WSS</SelectItem>
                <SelectItem value="auto">Auto</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="node_host">{{ t('nodes.nodeHost') }}</Label>
            <Input id="node_host" v-model="formValue.node_host" :placeholder="t('nodes.nodeHostPlaceholder')" />
          </div>
          <div class="space-y-2">
            <Label for="node_port">{{ t('nodes.nodePort') }}</Label>
            <Input id="node_port" v-model.number="formValue.node_port" type="number" :min="1" :max="65535" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="showAddModal = false">{{ t('nodes.cancel') }}</Button>
            <Button type="submit" :disabled="saving">{{ t('nodes.addNode') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showEditModal">
      <DialogContent class="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>{{ t('nodes.edit') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleUpdate" class="space-y-4">
          <div class="space-y-2">
            <Label for="edit-name">{{ t('nodes.name') }}</Label>
            <Input id="edit-name" v-model="editForm.name" />
          </div>
          <div class="space-y-2">
            <Label for="edit-host">{{ t('nodes.host') }}</Label>
            <Input id="edit-host" v-model="editForm.host" />
          </div>
          <div class="space-y-2">
            <Label for="edit-port">{{ t('nodes.port') }}</Label>
            <Input id="edit-port" v-model.number="editForm.port" type="number" :min="1" :max="65535" />
          </div>
          <div class="space-y-2">
            <Label for="edit-connect_mode">{{ t('nodes.connectMode') }}</Label>
            <Select v-model="editForm.connect_mode">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="active_tls">Active TLS</SelectItem>
                <SelectItem value="active_wss">Active WSS</SelectItem>
                <SelectItem value="passive_tls">Passive TLS</SelectItem>
                <SelectItem value="passive_wss">Passive WSS</SelectItem>
                <SelectItem value="auto">Auto</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="edit-node_host">{{ t('nodes.nodeHost') }}</Label>
            <Input id="edit-node_host" v-model="editForm.node_host" />
          </div>
          <div class="space-y-2">
            <Label for="edit-node_port">{{ t('nodes.nodePort') }}</Label>
            <Input id="edit-node_port" v-model.number="editForm.node_port" type="number" :min="1" :max="65535" />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="showEditModal = false">{{ t('nodes.cancel') }}</Button>
            <Button type="submit" :disabled="saving">{{ t('nodes.save') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, Refresh, Zap, Trash } from 'lucide-vue-next'
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

const { t } = useI18n()
const { toast } = useToast()
const loading = ref(true)
const saving = ref(false)
const data = ref<any[]>([])
const showAddModal = ref(false)
const showEditModal = ref(false)
const editingId = ref<number | null>(null)

const formValue = ref({ name: '', host: '', port: 18888, connect_mode: 'passive_tls', node_host: '', node_port: 18888 })
const editForm = ref({ name: '', host: '', port: 18888, connect_mode: 'passive_tls', node_host: '', node_port: 18888 })

async function loadData() {
  loading.value = true
  try {
    const res = await api.get('/nodes')
    data.value = res.data || []
  } catch {
    toast({ title: 'Error', description: t('nodes.failedToLoad'), variant: 'destructive' })
  } finally {
    loading.value = false
  }
}

async function handleAdd() {
  if (!formValue.value.name || !formValue.value.host) return

  saving.value = true
  try {
    await api.post('/nodes', formValue.value)
    toast({ title: t('nodes.nodeAdded') })
    showAddModal.value = false
    formValue.value = { name: '', host: '', port: 18888, connect_mode: 'passive_tls', node_host: '', node_port: 18888 }
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToAdd'), variant: 'destructive' })
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
    toast({ title: t('nodes.nodeUpdated') })
    showEditModal.value = false
    editingId.value = null
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToUpdate'), variant: 'destructive' })
  } finally {
    saving.value = false
  }
}

async function handleConnect(id: number) {
  try {
    const res = await api.post(`/nodes/${id}/connect`)
    toast({ title: res.data?.message || t('nodes.connectionInitiated') })
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToConnect'), variant: 'destructive' })
  }
}

async function handleDelete(id: number) {
  try {
    await api.delete(`/nodes/${id}`)
    toast({ title: t('nodes.nodeDeleted') })
    loadData()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToDelete'), variant: 'destructive' })
  }
}

onMounted(loadData)
</script>