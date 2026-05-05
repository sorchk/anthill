<script setup lang="ts">
import { DropdownMenuTrigger } from 'radix-vue'
import { cn } from '@/lib/utils'
</script>

<template>
  <DropdownMenuTrigger :class="cn('flex items-center', $attrs.class as string)" v-bind="$attrs">
    <slot />
  </DropdownMenuTrigger>
</template>
