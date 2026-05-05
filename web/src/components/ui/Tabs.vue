<script setup lang="ts">
import { TabsRoot, TabsList, TabsTrigger, TabsContent } from 'radix-vue'
import { cn } from '@/lib/utils'

const modelValue = defineModel<string>()
</script>

<template>
  <TabsRoot v-model="modelValue" :class="cn('data-[orientation=horizontal]:w-full', $attrs.class as string)">
    <slot />
  </TabsRoot>
</template>
