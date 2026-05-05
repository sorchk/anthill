<template>
  <div class="node-detail-page max-w-6xl">
    <div class="flex justify-between items-center mb-6">
      <div class="flex items-center gap-4">
        <Button variant="ghost" @click="$router.push('/nodes')">
          <ArrowLeft class="h-4 w-4" />
        </Button>
        <h2 class="text-2xl font-semibold">{{ node?.name || t('nodes.nodeDetail') }}</h2>
        <Badge :variant="statusVariant">{{ node?.status || 'unknown' }}</Badge>
      </div>
      <div class="flex gap-2">
        <Button @click="testConnection" :disabled="testing">
          <Zap class="h-4 w-4 mr-2" />
          {{ t('nodes.testConnection') }}
        </Button>
        <Button @click="showEditModal = true">
          <Pencil class="h-4 w-4 mr-2" />
          {{ t('nodes.edit') }}
        </Button>
      </div>
    </div>

    <Tabs v-model:value="activeTab">
      <TabsList>
        <TabsTrigger value="info">{{ t('nodes.info') }}</TabsTrigger>
        <TabsTrigger value="certificates">{{ t('nodes.certificates') }}</TabsTrigger>
        <TabsTrigger value="plugins">{{ t('nodes.plugins') }}</TabsTrigger>
        <TabsTrigger value="clients">{{ t('nodes.clients') }}</TabsTrigger>
        <TabsTrigger value="services">{{ t('nodes.services') }}</TabsTrigger>
        <TabsTrigger value="execute">{{ t('nodes.execute') }}</TabsTrigger>
      </TabsList>

      <TabsContent value="info">
        <Card v-if="node">
          <CardContent class="pt-6">
            <div class="grid grid-cols-2 gap-4 text-sm">
              <div><span class="text-muted-foreground">{{ t('nodes.name') }}:</span> {{ node.name }}</div>
              <div><span class="text-muted-foreground">{{ t('nodes.host') }}:</span> {{ node.host }}</div>
              <div><span class="text-muted-foreground">{{ t('nodes.port') }}:</span> {{ node.port }}</div>
              <div><span class="text-muted-foreground">{{ t('nodes.status') }}:</span> <Badge :variant="statusVariant">{{ node.status }}</Badge></div>
              <div><span class="text-muted-foreground">{{ t('nodes.connectMode') }}:</span> <Badge :variant="connectModeVariant">{{ node.connect_mode || 'passive_tls' }}</Badge></div>
              <div><span class="text-muted-foreground">{{ t('nodes.nodeHost') }}:</span> {{ node.node_host || '-' }}</div>
              <div><span class="text-muted-foreground">{{ t('nodes.nodePort') }}:</span> {{ node.node_port || 18888 }}</div>
              <div><span class="text-muted-foreground">{{ t('nodes.lastSeen') }}:</span> {{ node.last_seen ? new Date(node.last_seen).toLocaleString() : '-' }}</div>
              <div><span class="text-muted-foreground">{{ t('nodes.lastConnMode') }}:</span> {{ node.last_conn_mode || '-' }}</div>
              <div><span class="text-muted-foreground">{{ t('nodes.certExpires') }}:</span> {{ node.cert_expires ? new Date(node.cert_expires).toLocaleString() : '-' }}</div>
              <div><span class="text-muted-foreground">{{ t('nodes.created') }}:</span> {{ node.created_at ? new Date(node.created_at).toLocaleString() : '-' }}</div>
            </div>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="certificates">
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <span>{{ t('nodes.certificateManagement') }}</span>
              <div class="flex gap-2">
                <Button size="sm" variant="outline" @click="resetToken" :disabled="resettingToken">
                  {{ t('nodes.resetToken') }}
                </Button>
                <Button size="sm" variant="outline" @click="renewCert" :disabled="renewing">
                  {{ t('nodes.renewCert') }}
                </Button>
                <Button v-if="node?.cert_serial" size="sm" variant="destructive" @click="revokeCert" :disabled="revoking">
                  {{ t('nodes.revokeCert') }}
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div v-if="node" class="space-y-3 text-sm">
              <div><span class="text-muted-foreground">{{ t('nodes.certSerial') }}:</span> {{ node.cert_serial || '-' }}</div>
              <div><span class="text-muted-foreground">{{ t('nodes.certExpires') }}:</span> {{ node.cert_expires ? new Date(node.cert_expires).toLocaleString() : '-' }}</div>
              <div class="flex items-center gap-2">
                <span class="text-muted-foreground">{{ t('nodes.bootstrapToken') }}:</span>
                <code class="text-xs bg-muted px-1 rounded">{{ node.bootstrap_token ? node.bootstrap_token.substring(0, 8) + '...' : '-' }}</code>
                <Button v-if="node?.bootstrap_token" size="sm" variant="ghost" @click="copyToken">{{ t('nodes.copy') }}</Button>
              </div>
            </div>
            <Alert v-if="!node?.cert_serial" variant="destructive" class="mt-4">
              {{ t('nodes.noCertificate') }}
            </Alert>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="plugins">
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <span>{{ t('nodes.installedPlugins') }}: {{ nodePlugins.length }}</span>
              <Button size="sm" @click="showInstallPluginModal = true">
                <Plus class="h-4 w-4 mr-2" />
                {{ t('nodes.installPlugin') }}
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <Table v-if="nodePlugins.length > 0">
              <TableHeader>
                <TableRow>
                  <TableHead>{{ t('plugins.name') }}</TableHead>
                  <TableHead>{{ t('plugins.version') }}</TableHead>
                  <TableHead>{{ t('plugins.type') }}</TableHead>
                  <TableHead>{{ t('nodes.status') }}</TableHead>
                  <TableHead>{{ t('nodes.actions') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="row in nodePlugins" :key="row.name">
                  <TableCell>{{ row.name }}</TableCell>
                  <TableCell>{{ row.version }}</TableCell>
                  <TableCell>{{ row.type }}</TableCell>
                  <TableCell>
                    <Badge :variant="row.enabled ? 'default' : 'secondary'">{{ row.enabled ? t('nodes.enabled') : t('nodes.disabled') }}</Badge>
                  </TableCell>
                  <TableCell>
                    <div class="flex gap-2">
                      <Button variant="ghost" size="sm" @click="togglePlugin(row, !row.enabled)">
                        {{ row.enabled ? t('nodes.disable') : t('nodes.enable') }}
                      </Button>
                      <AlertDialog>
                        <AlertDialogTrigger as-child>
                          <Button variant="ghost" size="sm">
                            <Trash class="h-4 w-4 text-destructive" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>{{ t('nodes.uninstallConfirm') }}</AlertDialogTitle>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>{{ t('nodes.cancel') }}</AlertDialogCancel>
                            <AlertDialogAction @click="uninstallPlugin(row.name)">{{ t('nodes.uninstall') }}</AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
            <div v-else class="text-center py-8 text-muted-foreground">{{ t('nodes.noPlugins') }}</div>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="clients">
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <span>{{ t('nodes.clients') }}: {{ clients.length }}</span>
              <Button size="sm" @click="showAddClientModal = true">
                <Plus class="h-4 w-4 mr-2" />
                {{ t('nodes.addClient') }}
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <Table v-if="clients.length > 0">
              <TableHeader>
                <TableRow>
                  <TableHead>CN</TableHead>
                  <TableHead>{{ t('nodes.role') }}</TableHead>
                  <TableHead>{{ t('nodes.allowedPlugins') }}</TableHead>
                  <TableHead>{{ t('nodes.created') }}</TableHead>
                  <TableHead>{{ t('nodes.actions') }}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-for="row in clients" :key="row.id">
                  <TableCell>{{ row.client_cn }}</TableCell>
                  <TableCell>{{ row.role }}</TableCell>
                  <TableCell class="max-w-[200px] truncate">{{ row.allowed_plugins }}</TableCell>
                  <TableCell>{{ new Date(row.created_at).toLocaleString() }}</TableCell>
                  <TableCell>
                    <AlertDialog>
                      <AlertDialogTrigger as-child>
                        <Button variant="ghost" size="icon">
                          <Trash class="h-4 w-4 text-destructive" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>{{ t('nodes.deleteConfirm') }}</AlertDialogTitle>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel>{{ t('nodes.cancel') }}</AlertDialogCancel>
                          <AlertDialogAction @click="deleteClient(row.id)">{{ t('nodes.delete') }}</AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
            <div v-else class="text-center py-8 text-muted-foreground">{{ t('nodes.noClients') }}</div>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="services">
        <Card>
          <CardHeader>
            <div class="flex items-center justify-between">
              <span>{{ t('nodes.servicePlugins') }}</span>
              <Button variant="outline" size="sm" @click="loadServices">
                <Refresh class="h-4 w-4 mr-2" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div v-if="services.length > 0" class="space-y-4">
              <div v-for="svc in services" :key="svc.name" class="border rounded-lg p-4">
                <div class="flex items-center justify-between mb-2">
                  <div class="flex items-center gap-2">
                    <span class="font-medium">{{ svc.name }}</span>
                    <Badge :variant="svc.status === 'running' ? 'default' : 'secondary'">{{ svc.status }}</Badge>
                  </div>
                  <div class="flex gap-2">
                    <Button size="sm" variant="outline" :disabled="svc.status === 'running'" @click="controlService(svc.name, 'start')">{{ t('nodes.start') }}</Button>
                    <Button size="sm" variant="outline" :disabled="svc.status !== 'running'" @click="controlService(svc.name, 'stop')">{{ t('nodes.stop') }}</Button>
                    <Button size="sm" variant="outline" @click="controlService(svc.name, 'restart')">{{ t('nodes.restart') }}</Button>
                  </div>
                </div>
              </div>
            </div>
            <div v-else class="text-center py-8 text-muted-foreground">{{ t('nodes.noServices') }}</div>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="execute">
        <Card>
          <CardHeader>
            <CardTitle>{{ t('nodes.callPlugin') }}</CardTitle>
          </CardHeader>
          <CardContent class="space-y-4">
            <div class="space-y-2">
              <Label for="plugin">{{ t('nodes.selectPlugin') }}</Label>
              <Select v-model="executeForm.plugin">
                <SelectTrigger>
                  <SelectValue :placeholder="t('nodes.selectPlugin')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="p in pluginSelectOptions" :key="p.value" :value="p.value">{{ p.label }}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div class="space-y-2">
              <Label for="func">{{ t('nodes.function') }}</Label>
              <Input id="func" v-model="executeForm.function" placeholder="list" />
            </div>
            <div class="space-y-2">
              <Label for="args">{{ t('nodes.arguments') }}</Label>
              <Input id="args" v-model="executeForm.argsStr" placeholder="arg1, arg2" />
            </div>
            <Button @click="executePlugin" :disabled="executing">
              <Loader2 v-if="executing" class="mr-2 h-4 w-4 animate-spin" />
              {{ t('nodes.execute') }}
            </Button>

            <div v-if="executeResult" class="mt-6">
              <Separator class="my-4" />
              <h4 class="font-medium mb-2">{{ t('nodes.result') }}</h4>
              <pre class="bg-muted p-4 rounded text-xs overflow-auto max-h-[300px]">{{ executeResult }}</pre>
            </div>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>

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
            <Input id="edit-node_host" v-model="editForm.node_host" :placeholder="t('nodes.nodeHostPlaceholder')" />
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

    <Dialog v-model:open="showInstallPluginModal">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{{ t('nodes.installPlugin') }}</DialogTitle>
        </DialogHeader>
        <div class="space-y-4">
          <div class="space-y-2">
            <Label for="repo-plugin">{{ t('nodes.selectFromRepo') }}</Label>
            <Select v-model="installForm.plugin_id">
              <SelectTrigger>
                <SelectValue :placeholder="t('nodes.selectPlugin')" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem v-for="p in repoPluginOptions" :key="p.value" :value="p.value">{{ p.label }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" @click="showInstallPluginModal = false">{{ t('nodes.cancel') }}</Button>
          <Button @click="handleInstallPlugin" :disabled="installing">{{ t('nodes.install') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog v-model:open="showAddClientModal">
      <DialogContent class="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{{ t('nodes.addClient') }}</DialogTitle>
        </DialogHeader>
        <form @submit.prevent="handleAddClient" class="space-y-4">
          <div class="space-y-2">
            <Label for="client-cn">{{ t('nodes.clientCN') }}</Label>
            <Input id="client-cn" v-model="clientForm.client_cn" placeholder="client001" />
          </div>
          <div class="space-y-2">
            <Label for="client-role">{{ t('nodes.role') }}</Label>
            <Select v-model="clientForm.role">
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="admin">Admin</SelectItem>
                <SelectItem value="operator">Operator</SelectItem>
                <SelectItem value="viewer">Viewer</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="space-y-2">
            <Label for="client-plugins">{{ t('nodes.allowedPlugins') }}</Label>
            <Select v-model="clientForm.allowed_plugins" multiple>
              <SelectTrigger>
                <SelectValue placeholder="*" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="*">* (All)</SelectItem>
                <SelectItem value="shell">shell</SelectItem>
                <SelectItem value="file_transfer">file_transfer</SelectItem>
                <SelectItem value="terminal">terminal</SelectItem>
                <SelectItem value="proxy">proxy</SelectItem>
                <SelectItem value="plugin_mgr">plugin_mgr</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" @click="showAddClientModal = false">{{ t('nodes.cancel') }}</Button>
            <Button type="submit" :disabled="addingClient">{{ t('nodes.add') }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Zap, Pencil, Plus, Trash, Refresh, Loader2 } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import {Card, CardHeader, CardTitle, CardContent} from '@/components/ui'
import Badge from '@/components/ui/Badge.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Separator from '@/components/ui/Separator.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui'
import { AlertDialog, AlertDialogTrigger, AlertDialogContent, AlertDialogHeader, AlertDialogTitle, AlertDialogFooter, AlertDialogCancel, AlertDialogAction } from '@/components/ui'
import Alert from '@/components/ui/Alert.vue'
import { useToast } from '@/components/ui/useToast.ts'
import api from '@/api'

const route = useRoute()
const { t } = useI18n()
const { toast } = useToast()

const nodeId = computed(() => Number(route.params.id))

const node = ref<any>(null)
const nodePlugins = ref<any[]>([])
const clients = ref<any[]>([])
const services = ref<any[]>([])
const repoPlugins = ref<any[]>([])

const activeTab = ref('info')
const loadingPlugins = ref(false)
const loadingClients = ref(false)
const testing = ref(false)
const saving = ref(false)
const installing = ref(false)
const addingClient = ref(false)
const executing = ref(false)
const renewing = ref(false)
const revoking = ref(false)
const resettingToken = ref(false)

const showEditModal = ref(false)
const showInstallPluginModal = ref(false)
const showAddClientModal = ref(false)

const editForm = ref({
  name: '',
  host: '',
  port: 18888,
  connect_mode: 'passive_tls',
  node_host: '',
  node_port: 18888
})

const installForm = ref({ plugin_id: null as number | null })
const clientForm = ref({ client_cn: '', role: 'viewer', allowed_plugins: [] as string[] })

const executeForm = ref({ plugin: '', function: '', argsStr: '' })
const executeResult = ref('')

const statusVariant = computed(() => {
  if (node.value?.status === 'online') return 'default'
  if (node.value?.status === 'offline') return 'destructive'
  return 'secondary'
})

const connectModeVariant = computed(() => {
  const mode = node.value?.connect_mode || 'passive_tls'
  if (mode.startsWith('active')) return 'outline'
  if (mode === 'auto') return 'secondary'
  return 'secondary'
})

const pluginSelectOptions = computed(() =>
  nodePlugins.value.map((p: any) => ({ label: p.name, value: p.name }))
)

const repoPluginOptions = computed(() =>
  repoPlugins.value.map((p: any) => ({ label: `${p.name} - ${p.version}`, value: p.id }))
)

async function loadNode() {
  try {
    const res = await api.get(`/nodes/${nodeId.value}`)
    node.value = res.data
    editForm.value = {
      name: node.value.name,
      host: node.value.host,
      port: node.value.port,
      connect_mode: node.value.connect_mode || 'passive_tls',
      node_host: node.value.node_host || '',
      node_port: node.value.node_port || 18888
    }
  } catch {
    toast({ title: 'Error', description: t('nodes.failedToLoad'), variant: 'destructive' })
  }
}

async function loadNodePlugins() {
  loadingPlugins.value = true
  try {
    const res = await api.get(`/nodes/${nodeId.value}/plugins`)
    nodePlugins.value = res.data || []
  } catch {
    toast({ title: 'Error', description: t('nodes.failedToLoadPlugins'), variant: 'destructive' })
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
    toast({ title: 'Error', description: t('nodes.failedToLoadClients'), variant: 'destructive' })
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
    toast({ title: 'Error', description: t('nodes.failedToLoadServices'), variant: 'destructive' })
  }
}

async function testConnection() {
  testing.value = true
  try {
    await api.post(`/nodes/${nodeId.value}/connect`)
    toast({ title: t('nodes.connectionSuccess') })
    loadNode()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.connectionFailed'), variant: 'destructive' })
  } finally {
    testing.value = false
  }
}

async function handleUpdate() {
  saving.value = true
  try {
    await api.put(`/nodes/${nodeId.value}`, editForm.value)
    toast({ title: t('nodes.nodeUpdated') })
    showEditModal.value = false
    loadNode()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToUpdate'), variant: 'destructive' })
  } finally {
    saving.value = false
  }
}

async function handleInstallPlugin() {
  if (!installForm.value.plugin_id) {
    toast({ title: 'Error', description: t('nodes.selectPlugin'), variant: 'destructive' })
    return
  }
  installing.value = true
  try {
    await api.post(`/nodes/${nodeId.value}/plugins/install`, { plugin_id: installForm.value.plugin_id })
    toast({ title: t('nodes.pluginInstalled') })
    showInstallPluginModal.value = false
    installForm.value.plugin_id = null
    loadNodePlugins()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToInstall'), variant: 'destructive' })
  } finally {
    installing.value = false
  }
}

async function togglePlugin(plugin: any, enable: boolean) {
  try {
    const action = enable ? 'enable' : 'disable'
    await api.post(`/nodes/${nodeId.value}/plugins/${plugin.name}/${action}`)
    toast({ title: enable ? t('nodes.pluginEnabled') : t('nodes.pluginDisabled') })
    loadNodePlugins()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToToggle'), variant: 'destructive' })
  }
}

async function uninstallPlugin(name: string) {
  try {
    await api.delete(`/nodes/${nodeId.value}/plugins/${name}`)
    toast({ title: t('nodes.pluginUninstalled') })
    loadNodePlugins()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToUninstall'), variant: 'destructive' })
  }
}

async function handleAddClient() {
  if (!clientForm.value.client_cn) return
  addingClient.value = true
  try {
    await api.post(`/nodes/${nodeId.value}/clients`, clientForm.value)
    toast({ title: t('nodes.clientAdded') })
    showAddClientModal.value = false
    clientForm.value = { client_cn: '', role: 'viewer', allowed_plugins: [] }
    loadClients()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToAddClient'), variant: 'destructive' })
  } finally {
    addingClient.value = false
  }
}

async function deleteClient(id: number) {
  try {
    await api.delete(`/nodes/${nodeId.value}/clients/${id}`)
    toast({ title: t('nodes.clientDeleted') })
    loadClients()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToDeleteClient'), variant: 'destructive' })
  }
}

async function controlService(name: string, action: string) {
  try {
    await api.post(`/nodes/${nodeId.value}/services/${name}/${action}`)
    toast({ title: t('nodes.serviceActionSuccess') })
    loadServices()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToControlService'), variant: 'destructive' })
  }
}

async function executePlugin() {
  if (!executeForm.value.plugin || !executeForm.value.function) {
    toast({ title: 'Error', description: t('nodes.fillRequiredFields'), variant: 'destructive' })
    return
  }
  executing.value = true
  try {
    const args = executeForm.value.argsStr ? executeForm.value.argsStr.split(',').map(s => s.trim()) : []
    const res = await api.post(`/nodes/${nodeId.value}/execute`, {
      plugin: executeForm.value.plugin,
      function: executeForm.value.function,
      args
    })
    executeResult.value = JSON.stringify(res.data, null, 2)
  } catch (e: any) {
    executeResult.value = JSON.stringify({ error: e?.response?.data?.error || 'Failed' }, null, 2)
  } finally {
    executing.value = false
  }
}

async function renewCert() {
  renewing.value = true
  try {
    await api.post(`/nodes/${nodeId.value}/cert/renew`, { days: 365 })
    toast({ title: t('nodes.certRenewed') })
    loadNode()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToRenewCert'), variant: 'destructive' })
  } finally {
    renewing.value = false
  }
}

async function revokeCert() {
  if (!confirm(t('nodes.confirmRevokeCert'))) return
  revoking.value = true
  try {
    await api.post(`/nodes/${nodeId.value}/cert/revoke`)
    toast({ title: t('nodes.certRevoked') })
    loadNode()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToRevokeCert'), variant: 'destructive' })
  } finally {
    revoking.value = false
  }
}

async function resetToken() {
  if (!confirm(t('nodes.confirmResetToken'))) return
  resettingToken.value = true
  try {
    await api.post(`/nodes/${nodeId.value}/reset-token`)
    toast({ title: t('nodes.tokenReset') })
    loadNode()
  } catch (e: any) {
    toast({ title: 'Error', description: e?.response?.data?.error || t('nodes.failedToResetToken'), variant: 'destructive' })
  } finally {
    resettingToken.value = false
  }
}

function copyToken() {
  if (node.value?.bootstrap_token) {
    navigator.clipboard.writeText(node.value.bootstrap_token)
    toast({ title: t('nodes.tokenCopied') })
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