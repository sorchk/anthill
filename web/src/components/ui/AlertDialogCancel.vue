<script setup lang="ts">
import { AlertDialogCancel } from 'radix-vue'
import { cn } from '@/lib/utils'
</script>

<template>
  <AlertDialogCancel :class="cn('mt-2 inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium ring-offset-background transition-colors hover:bg-secondary sm:mt-0', $attrs.class as string)">
    <slot />
  </AlertDialogCancel>
</template>
