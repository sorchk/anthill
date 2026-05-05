<script setup lang="ts">
import { DialogPortal } from 'radix-vue'

defineProps<{ open?: boolean }>()
</script>

<template>
  <DialogPortal>
    <Transition
      enter-active-class="data-[state=open]:animate-in data-[state=closed]:animate-out data-[state-closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state-closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2"
      leave-active-class="data-[state=open]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state-closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2"
    >
      <div v-if="open" class="fixed inset-0 z-50 bg-black/80" @click="$emit('update:open', false)" />
    </Transition>
    <div class="fixed inset-0 z-50 flex items-center justify-center">
      <slot />
    </div>
  </DialogPortal>
</template>
