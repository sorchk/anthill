<script setup lang="ts">
import { ref } from 'vue'
import { SelectRoot, SelectTrigger, SelectValue, SelectContent, SelectItem, SelectItemIndicator, SelectItemText, SelectGroup, SelectLabel, SelectSeparator, SelectScrollUpButton, SelectScrollDownButton, SelectIcon } from 'radix-vue'
import { ChevronUp, ChevronDown } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const modelValue = defineModel<string | number | null>()
const placeholder = defineProps<{ placeholder?: string }>()

const open = ref(false)
</script>

<template>
  <SelectRoot v-model="modelValue" :open="open" @update:open="(o: boolean) => open = o">
    <SelectTrigger
      :class="cn(
        'flex h-9 w-full items-center justify-between whitespace-nowrap rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1',
        $attrs.class as string
      )"
    >
      <SelectValue :placeholder="placeholder">
        <slot name="SelectValue" />
      </SelectValue>
      <SelectIcon class="ml-2">
        <ChevronDown class="h-4 w-4 opacity-50" />
      </SelectIcon>
    </SelectTrigger>

    <SelectContent
      class="relative z-50 max-h-96 min-w-[8rem] overflow-hidden rounded-md border bg-popover text-popover-foreground shadow-md data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2"
    >
      <SelectScrollUpButton class="flex cursor-default items-center justify-center py-1">
        <ChevronUp class="h-4 w-4" />
      </SelectScrollUpButton>

      <SelectViewport class="p-1">
        <slot />
      </SelectViewport>

      <SelectScrollDownButton class="flex cursor-default items-center justify-center py-1">
        <ChevronDown class="h-4 w-4" />
      </SelectScrollDownButton>
    </SelectContent>
  </SelectRoot>
</template>
