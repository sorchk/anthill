<script setup lang="ts">
import { DropdownMenuRoot, DropdownMenuTrigger, DropdownMenuPortal, DropdownMenuContent, DropdownMenuItem, DropdownMenuItemIndicator, DropdownMenuSeparator, DropdownMenuLabel, DropdownMenuGroup, DropdownMenuCheckboxItem } from 'radix-vue'
import { cn } from '@/lib/utils'

const open = defineModel<boolean>('open', { default: false })
</script>

<template>
  <DropdownMenuRoot v-model:open="open">
    <slot />
  </DropdownMenuRoot>
</template>
