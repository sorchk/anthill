<template>
  <div class="flex min-h-screen bg-background">
    <aside class="hidden md:flex w-[180px] flex-col border-r border-border bg-card">
      <div class="flex h-14 items-center border-b border-border px-4">
        <Server class="h-6 w-6 mr-2" />
        <span class="font-semibold text-lg">Anthill</span>
      </div>
      <nav class="flex-1 p-2">
        <ul class="space-y-1">
          <li v-for="item in menuItems" :key="item.key">
            <button
              @click="handleMenuSelect(item.key)"
              :class="[
                'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                activeMenu === item.key
                  ? 'bg-primary text-primary-foreground'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
              ]"
            >
              <component :is="item.icon" class="h-4 w-4" />
              {{ item.label }}
            </button>
          </li>
        </ul>
      </nav>
    </aside>

    <div class="flex flex-1 flex-col">
      <header class="flex h-14 items-center justify-between border-b border-border px-6">
        <div class="flex items-center gap-4">
          <span class="text-sm font-medium md:hidden">Anthill</span>
        </div>
        <div class="flex items-center gap-2">
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="sm">
                {{ locale === 'zh-CN' ? '中文' : 'EN' }}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem @click="handleLangChange('zh-CN')">中文</DropdownMenuItem>
              <DropdownMenuItem @click="handleLangChange('en-US')">English</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button variant="ghost" size="sm">
                {{ authStore.user?.username }}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem @click="handleUserMenu('settings')">
                {{ t('nav.settings') }}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem @click="handleUserMenu('logout')">
                {{ t('login.signOut') }}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </header>

      <main class="flex-1 p-6">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  Server,
  LayoutDashboard,
  Cpu,
  Puzzle,
  FileText,
  Upload,
  Users,
  Clock,
  Network,
  Settings,
} from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui'
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

const menuItems = computed(() => [
  { key: 'dashboard', label: t('nav.dashboard'), icon: LayoutDashboard },
  { key: 'nodes', label: t('nav.nodes'), icon: Cpu },
  { key: 'plugins', label: t('nav.plugins'), icon: Puzzle },
  { key: 'audit', label: t('nav.audit'), icon: FileText },
  { key: 'deployments', label: t('nav.deployments'), icon: Upload },
  { key: 'users', label: t('nav.users'), icon: Users },
  { key: 'sessions', label: t('nav.sessions'), icon: Clock },
  { key: 'tunnels', label: t('nav.tunnels'), icon: Network },
  { key: 'settings', label: t('nav.settings'), icon: Settings }
])

function handleMenuSelect(key: string) {
  router.push(routePath[key] || '/')
}

function handleLangChange(lang: string) {
  locale.value = lang as 'zh-CN' | 'en-US'
  localStorage.setItem('locale', lang)
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