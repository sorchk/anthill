<script setup lang="ts">
import { SelectValue } from 'radix-vue'
</script>

<template>
  <SelectValue v-bind="$attrs">
    <slot />
  </SelectValue>
</template>
