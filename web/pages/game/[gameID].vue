<template>
  <div class="flex h-screen min-h-screen w-full flex-col space-y-2 p-2">
    <!-- Game Over Overlay -->
    <div
      v-if="state?.status === 'completed'"
      class="absolute inset-0 z-50 flex flex-col items-center justify-center bg-black/70">
      <h1 class="text-5xl font-bold text-white dark:text-indigo-400">
        {{ state.winner === sessionData?.user.id ? 'You Win!' : 'You Lose' }}
      </h1>
      <p class="mt-4 text-3xl text-white/90">
        {{ state.you.setsWon }} - {{ state.opponent?.setsWon || 0 }}
      </p>
      <p v-if="state.sets.length" class="mt-2 text-lg text-white/70">
        {{ setScores }}
      </p>
      <NuxtLink
        to="/home"
        class="mt-8 rounded-xl bg-white px-6 py-3 text-lg font-semibold dark:bg-black/70 dark:text-white">
        Back to Home
      </NuxtLink>
    </div>

    <!-- Waiting for confirmation Overlay -->
    <div
      v-if="state?.status === 'awaiting_confirmation'"
      class="absolute inset-0 z-50 flex flex-col items-center justify-center gap-4 bg-black/70 p-4 text-center">
      <template v-if="state.proposedBy === sessionData?.user.id">
        <h1 class="text-3xl font-bold text-white">Waiting for {{ state.opponent?.name }} to confirm</h1>
      </template>
      <template v-else>
        <h1 class="text-3xl font-bold text-white">
          {{ nameOf(state.proposedBy) }} says {{ nameOf(state.winner) }} won
        </h1>
        <p v-if="state.score" class="text-xl text-white/80">{{ state.score }}</p>
        <div class="flex gap-4">
          <Button @click="confirmResult">Confirm</Button>
          <Button variant="secondary" @click="rejectResult">Reject</Button>
        </div>
      </template>
    </div>

    <!-- Cancelled Overlay -->
    <div
      v-if="state?.status === 'cancelled'"
      class="absolute inset-0 z-50 flex flex-col items-center justify-center bg-black/70">
      <h1 class="text-5xl font-bold text-white">Match cancelled</h1>
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

    <div class="flex h-16 shrink-0 items-center justify-center gap-4 text-xl">
      <span class="text-sm text-muted-foreground">
        {{ state?.bestOf && state.bestOf > 1 ? `Bo${state.bestOf}` : '1 set' }}
      </span>
      <span v-if="state?.bestOf && state.bestOf > 1" class="font-semibold">
        {{ state.you.setsWon }} - {{ state.opponent?.setsWon || 0 }}
      </span>
      <span v-if="state?.isDeuce" class="font-semibold text-yellow-400">Deuce</span>
    </div>

    <button
      class="flex flex-1 items-center justify-center rounded-2xl bg-emerald-700"
      :disabled="state?.status !== 'ongoing'"
      @click="incrementScore">
      <span class="text-9xl text-white/90">{{ state?.you.score || 0 }}</span>
    </button>

    <div v-if="state?.status === 'ongoing'" class="flex shrink-0 flex-wrap items-center justify-center gap-2">
      <template v-if="picking">
        <Button @click="pick(state.you.uid)">I won</Button>
        <Button v-if="state.opponent" @click="pick(state.opponent.uid)">
          {{ state.opponent.name }} won
        </Button>
        <Button variant="ghost" @click="picking = false">Back</Button>
      </template>
      <template v-else>
        <Button variant="secondary" @click="picking = true">Pick winner</Button>
        <Button variant="outline" @click="askCancel">Cancel match</Button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
const { data: sessionData } = await authClient.getSession()
const route = useRoute()
const gameID = route.params.gameID as string
const { state, incrementScore, proposeWinner, confirmResult, rejectResult, cancelMatch } = await useGameState(gameID)

const picking = ref(false)

const pick = (uid: string) => {
  picking.value = false
  proposeWinner(uid)
}

const askCancel = () => {
  if (confirm('Cancel this match? No result is saved.')) cancelMatch()
}

const nameOf = (uid?: string) =>
  uid === state.value?.you.uid ? state.value?.you.name : state.value?.opponent?.name

// completed set scores from your point of view, e.g. "11-7, 9-11, 12-10"
const setScores = computed(() => {
  const you = sessionData?.user.id
  const opp = state.value?.opponent?.uid
  if (!you || !opp) return ''
  return (state.value?.sets ?? []).map(set => `${set[you]}-${set[opp]}`).join(', ')
})
</script>

<style lang="postcss" scoped></style>