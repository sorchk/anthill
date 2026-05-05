<script setup lang="ts">
import { DropdownMenuItem } from 'radix-vue'
import { cn } from '@/lib/utils'
</script>

<template>
  <DropdownMenuItem
    :class="cn(
      'relative flex cursor-default select-none items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none transition-colors focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 [&>svg]:size-4 [&>svg]:shrink-0',
      $attrs.class as string
    )"
    v-bind="$attrs"
  >
    <slot />
  </DropdownMenuItem>
</template>
