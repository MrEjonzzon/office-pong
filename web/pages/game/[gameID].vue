<template>
  <div class="flex h-screen min-h-screen w-full flex-col space-y-2 p-2">
    <!-- Game Over Overlay -->
    <div
      v-if="state?.status === 'finished'"
      class="absolute inset-0 z-50 flex flex-col items-center justify-center bg-black/70">
      <h1 class="text-5xl font-bold text-white dark:text-indigo-400">
        {{ state.winner === sessionData?.user.id ? 'You Win!' : 'You Lose' }}
      </h1>
      <p class="mt-4 text-xl text-white/80">
        {{ state.you.score }} - {{ state.opponent?.score || 0 }}
      </p>
      <NuxtLink
        to="/home"
        class="mt-8 rounded-xl bg-white px-6 py-3 text-lg font-semibold dark:bg-black/70 dark:text-white">
        Back to Home
      </NuxtLink>
    </div>

    <button
      class="flex flex-1 items-center justify-center rounded-2xl bg-rose-700"
      :class="!state?.opponent ? '' : 'bg-rose-700'"
      :disabled="state?.status === 'finished'"
      @click="console.log('TODO: local game logic')">
      <span class="text-9xl text-white/90">
        {{ state?.opponent?.score || 0 }}
      </span>
    </button>

    <div class="flex h-16 shrink-0 justify-center text-2xl">
      <div class="flex h-full items-center space-x-6 px-4">
        <Icon name="lucide:flag" class="text-red-500" />
        <Icon name="lucide:bell-ring" class="text-yellow-400" />
      </div>
    </div>

    <button
      class="flex flex-1 items-center justify-center rounded-2xl bg-emerald-700"
      :disabled="state?.status === 'finished'"
      @click="incrementScore">
      <span class="text-9xl text-white/90">{{ state?.you.score || 0 }}</span>
    </button>
  </div>
</template>

<script setup lang="ts">
const { data: sessionData } = await authClient.getSession()
const route = useRoute()
const gameID = route.params.gameID as string
const { state, incrementScore } = await useGameState(gameID)
</script>

<style lang="postcss" scoped></style>