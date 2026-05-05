<script setup lang="ts">
import { AlertDialogAction } from 'radix-vue'
import { cn } from '@/lib/utils'
</script>

<template>
  <AlertDialogAction :class="cn('inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium ring-offset-background transition-colors hover:bg-primary/90 sm:inline-flex', $attrs.class as string)">
    <slot />
  </AlertDialogAction>
</template>
