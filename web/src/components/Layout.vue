<template>
  <n-layout class="app-layout">
    <n-layout-header class="app-header">
      <div class="header-content">
        <div class="header-left">
          <n-icon size="24"><server-outline /></n-icon>
          <span class="app-title">Anthill</span>
        </div>
        <div class="header-right">
          <n-dropdown :options="langOptions" @select="handleLangChange">
            <n-button quaternary size="small" style="margin-right: 8px">
              <template #icon><n-icon><language-outline /></n-icon></template>
              {{ locale === 'zh-CN' ? '中文' : 'EN' }}
            </n-button>
          </n-dropdown>
          <n-dropdown :options="userMenuOptions" @select="handleUserMenu">
            <n-button quaternary>
              <template #icon><n-icon><person-circle-outline /></n-icon></template>
              {{ authStore.user?.username }}
            </n-button>
          </n-dropdown>
        </div>
      </div>
    </n-layout-header>

    <n-layout has-sider class="app-body">
      <n-layout-sider
        bordered
        :width="180"
        :native-scrollbar="false"
        class="app-sider"
      >
        <n-menu
          :value="activeMenu"
          :options="menuOptions"
          @update:value="handleMenuSelect"
        />
      </n-layout-sider>

      <n-layout-content class="app-content">
        <router-view />
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, h, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  NLayout, NLayoutHeader, NLayoutSider, NLayoutContent,
  NMenu, NButton, NIcon, NDropdown, MenuOption
} from 'naive-ui'
import {
  ServerOutline, PersonCircleOutline, GridOutline,
  HardwareChipOutline, ExtensionPuzzleOutline, DocumentTextOutline, CloudUploadOutline,
  PeopleOutline, TimeOutline, LanguageOutline, SettingsOutline, GitNetworkOutline
} from '@vicons/ionicons5'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const { t, locale } = useI18n()
const authStore = useAuthStore()

const activeMenu = computed(() => {
  const map: Record<string, string> = {
    '/': 'dashboard',
    '/nodes': 'nodes',
    '/plugins': 'plugins',
    '/audit': 'audit',
    '/deployments': 'deployments',
    '/users': 'users',
    '/sessions': 'sessions',
    '/tunnels': 'tunnels',
    '/settings': 'settings'
  }
  return map[route.path] || 'dashboard'
})

const routePath: Record<string, string> = {
  'dashboard': '/',
  'nodes': '/nodes',
  'plugins': '/plugins',
  'audit': '/audit',
  'deployments': '/deployments',
  'users': '/users',
  'sessions': '/sessions',
  'tunnels': '/tunnels',
  'settings': '/settings'
}

function makeMenuOptions(): MenuOption[] {
  return [
    { label: t('nav.dashboard'), key: 'dashboard', icon: () => h(NIcon, null, { default: () => h(GridOutline) }) },
    { label: t('nav.nodes'), key: 'nodes', icon: () => h(NIcon, null, { default: () => h(HardwareChipOutline) }) },
    { label: t('nav.plugins'), key: 'plugins', icon: () => h(NIcon, null, { default: () => h(ExtensionPuzzleOutline) }) },
    { label: t('nav.audit'), key: 'audit', icon: () => h(NIcon, null, { default: () => h(DocumentTextOutline) }) },
    { label: t('nav.deployments'), key: 'deployments', icon: () => h(NIcon, null, { default: () => h(CloudUploadOutline) }) },
    { label: t('nav.users'), key: 'users', icon: () => h(NIcon, null, { default: () => h(PeopleOutline) }) },
    { label: t('nav.sessions'), key: 'sessions', icon: () => h(NIcon, null, { default: () => h(TimeOutline) }) },
    { label: t('nav.tunnels'), key: 'tunnels', icon: () => h(NIcon, null, { default: () => h(GitNetworkOutline) }) },
    { label: t('nav.settings'), key: 'settings', icon: () => h(NIcon, null, { default: () => h(SettingsOutline) }) }
  ]
}

const menuOptions = ref(makeMenuOptions())

function handleMenuSelect(key: string) {
  router.push(routePath[key] || '/')
}

const langOptions = [
  { label: '中文', key: 'zh-CN' },
  { label: 'English', key: 'en-US' }
]

const userMenuOptions = [
  { label: t('nav.settings'), key: 'settings' },
  { type: 'divider', key: 'd1' },
  { label: t('login.signOut'), key: 'logout' }
]

function handleLangChange(key: string) {
  locale.value = key as 'zh-CN' | 'en-US'
  localStorage.setItem('locale', key)
  menuOptions.value = makeMenuOptions()
}

function handleUserMenu(key: string) {
  if (key === 'settings') {
    router.push('/settings')
  } else if (key === 'logout') {
    authStore.logout()
    router.push('/login')
  }
}
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
}

.app-header {
  height: 60px;
  padding: 0 24px;
  display: flex;
  align-items: center;
}

.header-content {
  width: 100%;
  display: flex;
  justify-content: space-between;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.app-title {
  font-size: 18px;
  font-weight: 600;
}

.header-right {
  display: flex;
  align-items: center;
}

.app-body {
  height: calc(100vh - 60px);
}

.app-sider {
  background: #fff;
}

.app-content {
  padding: 24px;
}
</style>