import { createRouter, createWebHistory, RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes: RouteRecordRaw[] = [
  {
    path: '/init',
    name: 'Init',
    component: () => import('@/views/Init.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/components/Layout.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', name: 'Dashboard', component: () => import('@/views/Dashboard.vue') },
      { path: 'nodes', name: 'Nodes', component: () => import('@/views/Nodes.vue') },
      { path: 'nodes/:id', name: 'NodeDetail', component: () => import('@/views/NodeDetail.vue') },
      { path: 'plugins', name: 'Plugins', component: () => import('@/views/Plugins.vue') },
      { path: 'audit', name: 'Audit', component: () => import('@/views/Audit.vue'), meta: { requiresAdmin: true } },
      { path: 'deployments', name: 'Deployments', component: () => import('@/views/Deployments.vue'), meta: { requiresAuth: true, requiresAdmin: true } },
      { path: 'users', name: 'Users', component: () => import('@/views/Users.vue'), meta: { requiresAuth: true, requiresAdmin: true } },
      { path: 'sessions', name: 'Sessions', component: () => import('@/views/Sessions.vue'), meta: { requiresAuth: true } },
      { path: 'tunnels', name: 'Tunnels', component: () => import('@/views/Tunnels.vue'), meta: { requiresAuth: true } },
      { path: 'settings', name: 'Settings', component: () => import('@/views/Settings.vue'), meta: { requiresAuth: true } }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()

  if (to.name === 'Init') {
    const initialized = await authStore.checkInitialized()
    if (initialized) {
      next({ name: 'Login' })
    } else {
      next()
    }
    return
  }

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    const initialized = await authStore.checkInitialized()
    if (!initialized) {
      next({ name: 'Init' })
    } else {
      next({ name: 'Login' })
    }
  } else if (to.name === 'Login' && authStore.isAuthenticated) {
    next({ name: 'Dashboard' })
  } else {
    next()
  }
})

export default router