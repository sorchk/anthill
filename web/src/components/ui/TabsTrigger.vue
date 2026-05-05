<script setup lang="ts">
import { TabsTrigger } from 'radix-vue'
import { cn } from '@/lib/utils'

const props = defineProps<{ value: string }>()
</script>

<template>
  <TabsTrigger
    :value="props.value"
    :class="cn(
      'inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow',
      $attrs.class as string
    )"
  >
    <slot />
  </TabsTrigger>
</template>
