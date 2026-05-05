<script setup lang="ts">
import { Separator } from 'radix-vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ orientation?: 'horizontal' | 'vertical'; class?: string }>()
</script>

<template>
  <Separator
    :class="cn(
      'shrink-0 bg-border',
      props.orientation === 'vertical' ? 'w-px h-full' : 'h-px w-full',
      props.class
    )"
    v-bind="$attrs"
  />
</template>
