<template>
  <div class="dashboard">
    <h2 class="page-title">{{ t('dashboard.title') }}</h2>

    <n-grid :cols="4" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
      <n-gi span="4 m:2 l:1">
        <n-card class="stat-card" bordered>
          <div class="stat-content">
            <div class="stat-icon nodes">
              <n-icon size="32"><hardware-chip-outline /></n-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.nodes }}</div>
              <div class="stat-label">{{ t('dashboard.totalNodes') }}</div>
            </div>
          </div>
        </n-card>
      </n-gi>

      <n-gi span="4 m:2 l:1">
        <n-card class="stat-card" bordered>
          <div class="stat-content">
            <div class="stat-icon online">
              <n-icon size="32"><checkmark-circle /></n-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.online }}</div>
              <div class="stat-label">{{ t('dashboard.onlineNodes') }}</div>
            </div>
          </div>
        </n-card>
      </n-gi>

      <n-gi span="4 m:2 l:1">
        <n-card class="stat-card" bordered>
          <div class="stat-content">
            <div class="stat-icon plugins">
              <n-icon size="32"><ExtensionPuzzleOutline /></n-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.plugins }}</div>
              <div class="stat-label">{{ t('dashboard.totalPlugins') }}</div>
            </div>
          </div>
        </n-card>
      </n-gi>

      <n-gi span="4 m:2 l:1">
        <n-card class="stat-card" bordered>
          <div class="stat-content">
            <div class="stat-icon audit">
              <n-icon size="32"><document-text-outline /></n-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.auditLogs }}</div>
              <div class="stat-label">{{ t('dashboard.auditLogs') }}</div>
            </div>
          </div>
        </n-card>
      </n-gi>
    </n-grid>

    <n-card :title="t('dashboard.recentActivity')" class="activity-card" bordered style="margin-top: 24px">
      <n-list v-if="recentLogs.length > 0">
        <n-list-item v-for="log in recentLogs" :key="log.id">
          <n-thing :title="log.action" :description="`${log.username} · ${formatTime(log.created_at)}`">
            <template #avatar>
              <n-tag :type="log.status < 400 ? 'success' : 'error'" size="small">
                {{ log.status }}
              </n-tag>
            </template>
          </n-thing>
        </n-list-item>
      </n-list>
      <n-empty v-else :description="t('dashboard.noActivity')" />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NGrid, NGi, NCard, NIcon, NList, NListItem, NThing, NTag, NEmpty
} from 'naive-ui'
import {
  HardwareChipOutline, CheckmarkCircle,
  ExtensionPuzzleOutline, DocumentTextOutline
} from '@vicons/ionicons5'
import api from '@/api'

const { t } = useI18n()

const stats = ref({ nodes: 0, online: 0, plugins: 0, auditLogs: 0 })
const recentLogs = ref<any[]>([])

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

<style scoped>
.dashboard {
  max-width: 1200px;
}

.page-title {
  margin: 0 0 24px 0;
  font-size: 24px;
  font-weight: 600;
}

.stat-card {
  text-align: center;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.stat-icon.nodes { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
.stat-icon.online { background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%); }
.stat-icon.plugins { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.stat-icon.audit { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); }

.stat-info {
  text-align: left;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #1a1a1a;
  line-height: 1.2;
}

.stat-label {
  font-size: 14px;
  color: #666;
}

.activity-card {
  margin-top: 24px;
}
</style>