<script setup lang="ts">
import { cn } from '@/lib/utils'
</script>

<template>
  <header :class="cn('flex flex-col space-y-1.5 text-center sm:text-left', $attrs.class as string)">
    <slot />
  </header>
</template>
