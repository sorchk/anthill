<template>
  <div class="audit-page max-w-7xl">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-semibold">{{ t('audit.title') }}</h2>
      <div class="flex items-center gap-2">
        <Button variant="outline" @click="handleExport">
          <Download class="h-4 w-4 mr-2" />
          {{ t('audit.export') }}
        </Button>
        <Select v-model="filters.action" :placeholder="t('audit.filterByAction')" clearable class="w-[200px]" @update:modelValue="loadData">
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="GET">GET</SelectItem>
            <SelectItem value="POST">POST</SelectItem>
            <SelectItem value="PUT">PUT</SelectItem>
            <SelectItem value="DELETE">DELETE</SelectItem>
          </SelectContent>
        </Select>
        <Select v-model="filters.username" :placeholder="t('audit.filterByUser')" clearable filterable class="w-[150px]" @update:modelValue="loadData">
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="user in userOptions" :key="user.value" :value="user.value">{{ user.label }}</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" size="icon" @click="loadData">
          <Refresh class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{{ t('audit.id') }}</TableHead>
              <TableHead>{{ t('audit.user') }}</TableHead>
              <TableHead>{{ t('audit.action') }}</TableHead>
              <TableHead>{{ t('audit.method') }}</TableHead>
              <TableHead>{{ t('audit.path') }}</TableHead>
              <TableHead>{{ t('audit.ip') }}</TableHead>
              <TableHead>{{ t('audit.status') }}</TableHead>
              <TableHead>{{ t('audit.time') }}</TableHead>
              <TableHead></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in data" :key="row.id">
              <TableCell>{{ row.id }}</TableCell>
              <TableCell>{{ row.username }}</TableCell>
              <TableCell>{{ row.action }}</TableCell>
              <TableCell>
                <Badge :variant="getMethodVariant(row.method)">{{ row.method }}</Badge>
              </TableCell>
              <TableCell class="max-w-[200px] truncate">{{ row.path }}</TableCell>
              <TableCell>{{ row.ip }}</TableCell>
              <TableCell>
                <Badge :variant="row.status < 400 ? 'default' : 'destructive'">{{ row.status }}</Badge>
              </TableCell>
              <TableCell>{{ formatDate(row.created_at) }}</TableCell>
              <TableCell>
                <Button variant="ghost" size="sm" @click="openDetail(row)">{{ t('audit.view') }}</Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <div class="flex items-center justify-between mt-4">
      <div class="text-sm text-muted-foreground">
        {{ t('audit.total', { count: total }) }}
      </div>
      <div class="flex items-center gap-2">
        <Select v-model="pagination.pageSize" class="w-[150px]" @update:modelValue="loadData">
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem :value="25">25 / page</SelectItem>
            <SelectItem :value="50">50 / page</SelectItem>
            <SelectItem :value="100">100 / page</SelectItem>
            <SelectItem :value="200">200 / page</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" :disabled="pagination.page <= 1" @click="handlePageChange(pagination.page - 1)">
          <ChevronLeft class="h-4 w-4" />
        </Button>
        <span class="text-sm">{{ pagination.page }}</span>
        <Button variant="outline" size="sm" :disabled="!hasMore" @click="handlePageChange(pagination.page + 1)">
          <ChevronRight class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <Dialog v-model:open="showDetailModal">
      <DialogContent class="sm:max-w-[600px]">
        <DialogHeader>
          <DialogTitle>Audit Log #{{ selectedLog?.id }}</DialogTitle>
        </DialogHeader>
        <div v-if="selectedLog" class="space-y-3">
          <div class="grid grid-cols-2 gap-2 text-sm">
            <div><span class="text-muted-foreground">{{ t('audit.id') }}:</span> {{ selectedLog.id }}</div>
            <div><span class="text-muted-foreground">{{ t('audit.user') }}:</span> {{ selectedLog.username }} ({{ selectedLog.user_id }})</div>
            <div><span class="text-muted-foreground">{{ t('audit.action') }}:</span> {{ selectedLog.action }}</div>
            <div><span class="text-muted-foreground">{{ t('audit.method') }}:</span> <Badge :variant="getMethodVariant(selectedLog.method)">{{ selectedLog.method }}</Badge></div>
            <div><span class="text-muted-foreground">{{ t('audit.path') }}:</span> {{ selectedLog.path }}</div>
            <div><span class="text-muted-foreground">{{ t('audit.ip') }}:</span> {{ selectedLog.ip }}</div>
            <div><span class="text-muted-foreground">{{ t('audit.status') }}:</span> <Badge :variant="selectedLog.status < 400 ? 'default' : 'destructive'">{{ selectedLog.status }}</Badge></div>
            <div><span class="text-muted-foreground">{{ t('audit.time') }}:</span> {{ formatDate(selectedLog.created_at) }}</div>
          </div>
          <div v-if="selectedLog.details" class="mt-4">
            <span class="text-muted-foreground">{{ t('audit.details') }}:</span>
            <pre class="mt-2 p-2 bg-muted rounded text-xs overflow-auto max-h-[200px]">{{ selectedLog.details }}</pre>
          </div>
        </div>
        <DialogFooter>
          <Button @click="showDetailModal = false">{{ t('audit.close') }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Download, Refresh, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import { Card, CardContent } from '@/components/ui'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui'
import Badge from '@/components/ui/Badge.vue'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui'
import { useToast } from '@/components/ui/useToast.ts'
import api from '@/api'

const { t } = useI18n()
const { toast } = useToast()

const loading = ref(true)
const data = ref<any[]>([])
const selectedLog = ref<any>(null)
const showDetailModal = ref(false)
const filters = ref<{ action: string | null; username: string | null }>({ action: null, username: null })
const userOptions = ref<any[]>([])
const total = ref(0)
const hasMore = ref(false)

const pagination = ref({
  page: 1,
  pageSize: 50
})

function getMethodVariant(method: string): 'default' | 'destructive' | 'outline' {
  const variants: Record<string, 'default' | 'destructive' | 'outline'> = {
    GET: 'outline',
    POST: 'default',
    PUT: 'outline',
    DELETE: 'destructive'
  }
  return variants[method] || 'outline'
}

function formatDate(date: string) {
  return new Date(date).toLocaleString()
}

function openDetail(log: any) {
  selectedLog.value = log
  showDetailModal.value = true
}

function handleExport() {
  window.open(`/api/audit/export?format=csv`, '_blank')
}

async function loadData() {
  loading.value = true
  try {
    const params: any = {
      page: pagination.value.page,
      page_size: pagination.value.pageSize
    }
    if (filters.value.action) params.action = filters.value.action
    if (filters.value.username) params.username = filters.value.username

    const res = await api.get('/audit', { params })
    data.value = res.data?.data || []
    total.value = res.data?.total || 0
    hasMore.value = data.value.length === pagination.value.pageSize

    const userSet = new Set<string>()
    data.value.forEach((log: any) => userSet.add(log.username))
    userOptions.value = Array.from(userSet).map(u => ({ label: u, value: u }))
  } catch {
    toast({ title: 'Error', description: t('audit.failedToLoad'), variant: 'destructive' })
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.value.page = page
  loadData()
}

onMounted(loadData)
</script>