<template>
  <div class="container flex flex-col gap-y-4 self-start px-4">
    <h1 class="self-center pb-4 text-2xl">Home</h1>

    <ViewsUserList />
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: 'logged-in',
})
</script>

<style lang="postcss" scoped></style>
