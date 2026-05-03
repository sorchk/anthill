import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export const useLocaleStore = defineStore('locale', () => {
  const locale = ref(localStorage.getItem('locale') || 'zh-CN')

  function setLocale(newLocale: 'zh-CN' | 'en-US') {
    locale.value = newLocale
    localStorage.setItem('locale', newLocale)
  }

  watch(locale, (newVal) => {
    localStorage.setItem('locale', newVal)
  })

  return { locale, setLocale }
})