import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN.json'
import enUS from './en-US.json'

type MessageSchema = typeof zhCN

function getDefaultLocale(): string {
  if (typeof window === 'undefined') return 'zh-CN'

  const stored = localStorage.getItem('locale')
  if (stored && ['zh-CN', 'en-US'].includes(stored)) {
    return stored
  }

  const browserLang = navigator.language || (navigator as any).userLanguage
  if (browserLang) {
    if (browserLang.toLowerCase().startsWith('zh')) {
      return 'zh-CN'
    }
    if (browserLang.toLowerCase().startsWith('en')) {
      return 'en-US'
    }
  }

  return 'zh-CN'
}

const i18n = createI18n<[MessageSchema], 'zh-CN' | 'en-US'>({
  legacy: false,
  locale: getDefaultLocale(),
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS
  }
})

export default i18n