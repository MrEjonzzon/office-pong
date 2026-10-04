<script lang="ts" setup>
import Toaster from '@/components/ui/toast/Toaster.vue'
import { VitePwaManifest } from '#components'
</script>
<template>
  <VitePwaManifest />
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>

  <ClientOnly>
    <Toaster />
  </ClientOnly>
</template>
