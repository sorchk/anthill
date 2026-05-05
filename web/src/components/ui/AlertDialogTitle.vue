<script setup lang="ts">
import { AlertDialogTitle } from 'radix-vue'
import { cn } from '@/lib/utils'
</script>

<template>
  <AlertDialogTitle :class="cn('text-lg font-semibold', $attrs.class as string)">
    <slot />
  </AlertDialogTitle>
</template>
