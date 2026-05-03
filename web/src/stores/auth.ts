import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref<{ id: number; username: string; role: string } | null>(null)

  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function init() {
    if (token.value) {
      axios.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
    }
  }

  async function login(username: string, password: string) {
    const response = await axios.post('/api/auth/login', { username, password })
    token.value = response.data.token
    user.value = response.data.user
    localStorage.setItem('token', token.value)
    axios.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
    return response.data
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
    delete axios.defaults.headers.common['Authorization']
  }

  async function fetchUser() {
    if (!token.value) return null
    try {
      axios.defaults.headers.common['Authorization'] = `Bearer ${token.value}`
      const response = await axios.get('/api/me')
      user.value = response.data
      return user.value
    } catch {
      logout()
      return null
    }
  }

  init()

  async function checkInitialized(): Promise<boolean> {
    try {
      const response = await axios.get('/api/auth/check')
      return response.data.initialized
    } catch {
      return true
    }
  }

  async function initAdmin(username: string, password: string, confirmPassword: string) {
    const response = await axios.post('/api/auth/init', {
      username,
      password,
      confirmPassword
    })
    return response.data
  }

  return { token, user, isAuthenticated, isAdmin, login, logout, fetchUser, checkInitialized, initAdmin }
})
