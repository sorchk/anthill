<template>
  <div class="max-w-6xl">
    <h2 class="text-2xl font-semibold mb-6">{{ t('dashboard.title') }}</h2>

    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
      <Card v-for="(stat, index) in statsCards" :key="index">
        <CardHeader class="flex flex-row items-center justify-between pb-2">
          <CardTitle class="text-sm font-medium">{{ stat.label }}</CardTitle>
          <component :is="stat.icon" class="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">{{ stat.value }}</div>
        </CardContent>
      </Card>
    </div>

    <Card>
      <CardHeader>
        <CardTitle>{{ t('dashboard.recentActivity') }}</CardTitle>
      </CardHeader>
      <CardContent>
        <div v-if="recentLogs.length > 0" class="space-y-4">
          <div v-for="log in recentLogs" :key="log.id" class="flex items-center justify-between border-b border-border pb-4 last:border-0">
            <div class="flex items-center gap-3">
              <Badge :variant="log.status < 400 ? 'default' : 'destructive'">
                {{ log.status }}
              </Badge>
              <div>
                <p class="font-medium">{{ log.action }}</p>
                <p class="text-sm text-muted-foreground">{{ log.username }} · {{ formatTime(log.created_at) }}</p>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="text-center py-8 text-muted-foreground">
          {{ t('dashboard.noActivity') }}
        </div>
      </CardContent>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Cpu, CheckCircle, Puzzle, FileText } from 'lucide-vue-next'
import {Card, CardHeader, CardTitle, CardContent} from '@/components/ui'
import Badge from '@/components/ui/Badge.vue'
import api from '@/api'

const { t } = useI18n()

const stats = ref({ nodes: 0, online: 0, plugins: 0, auditLogs: 0 })
const recentLogs = ref<any[]>([])

const statsCards = computed(() => [
  { label: t('dashboard.totalNodes'), value: stats.value.nodes, icon: Cpu },
  { label: t('dashboard.onlineNodes'), value: stats.value.online, icon: CheckCircle },
  { label: t('dashboard.totalPlugins'), value: stats.value.plugins, icon: Puzzle },
  { label: t('dashboard.auditLogs'), value: stats.value.auditLogs, icon: FileText }
])

function formatTime(time: string) {
  const d = new Date(time)
  return d.toLocaleString()
}

onMounted(async () => {
  try {
    const [nodesRes, pluginsRes, auditRes] = await Promise.all([
      api.get('/nodes'),
      api.get('/plugins'),
      api.get('/audit?page=1&page_size=5')
    ])

    const nodes = nodesRes.data || []
    stats.value = {
      nodes: nodes.length,
      online: nodes.filter((n: any) => n.status === 'online').length,
      plugins: (pluginsRes.data || []).length,
      auditLogs: auditRes.data?.total || 0
    }

    recentLogs.value = auditRes.data?.data || []
  } catch (e) {
    console.error('Failed to load dashboard:', e)
  }
})
</script>