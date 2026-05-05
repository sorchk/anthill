<template>
  <router-view />
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { locale } = useI18n()

watch(locale, (newLocale) => {
  document.documentElement.setAttribute('lang', newLocale === 'zh-CN' ? 'zh' : 'en')
}, { immediate: true })
</script>