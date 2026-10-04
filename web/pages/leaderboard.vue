<template>
  <div class="container flex flex-col gap-y-4 self-start px-4">
    <h1 class="self-center pb-2 text-2xl">Leaderboard</h1>

    <!-- skellington -->
    <Card>
      <div class="divide-y divide-border">
        <template v-if="pending">
          <div v-for="i in 10" :key="i" class="flex animate-pulse items-center gap-4 px-4 py-3">
            <div class="w-8 shrink-0">
              <div class="mx-auto h-5 w-5 rounded bg-muted" />
            </div>
            <div class="size-9 shrink-0 rounded-full bg-muted" />
            <div class="flex-1">
              <div class="h-4 w-28 rounded bg-muted" />
            </div>
            <div class="flex shrink-0 items-center gap-4">
              <div class="hidden gap-3 sm:flex">
                <div class="h-3 w-7 rounded bg-muted" />
                <div class="h-3 w-7 rounded bg-muted" />
              </div>
              <div class="flex flex-col items-end gap-1">
                <div class="h-4 w-10 rounded bg-muted" />
                <div class="h-3 w-6 rounded bg-muted" />
              </div>
            </div>
          </div>
        </template>

        <!-- leader board entries -->
        <template v-else>
          <div
            v-for="(entry, index) in entries"
            :key="index"
            class="flex items-center gap-4 px-4 py-3 transition-colors"
            :class="
              entry?.userId === sessionData?.user.id
                ? 'bg-primary/20 font-semibold'
                : entry && index < 3
                  ? 'bg-muted/40'
                  : 'hover:bg-muted/20'
            ">
            <div class="w-8 shrink-0 text-center">
              <span v-if="index === 0" class="text-2xl leading-none">🥇</span>
              <span v-else-if="index === 1" class="text-2xl leading-none">🥈</span>
              <span v-else-if="index === 2" class="text-2xl leading-none">🥉</span>
              <span v-else class="text-sm font-medium text-muted-foreground">{{ index + 1 }}</span>
            </div>

            <template v-if="entry">
              <NuxtImg v-if="entry.image" :src="entry.image" class="size-9 shrink-0 rounded-full" />
              <div v-else class="size-9 shrink-0 rounded-full bg-muted" />
              <span class="flex-1 truncate font-medium">{{ entry.name }}</span>
              <div class="flex shrink-0 items-center gap-4">
                <div class="hidden gap-3 text-xs text-muted-foreground sm:flex">
                  <span>{{ entry.wins }}W</span>
                  <span>{{ entry.losses }}L</span>
                </div>
                <div class="flex flex-col items-end">
                  <span class="text-sm font-semibold">{{ entry.mmr }}</span>
                  <span class="text-xs text-muted-foreground">MMR</span>
                </div>
              </div>
            </template>

            <!-- filling out if we have less than 10 on the leaderboard -->
            <template v-else>
              <div class="size-9 shrink-0 rounded-full border-2 border-dashed border-muted" />
              <span class="flex-1 text-sm text-muted-foreground/40">—</span>
              <div class="flex shrink-0 items-center gap-4">
                <div class="hidden gap-3 text-xs text-muted-foreground/40 sm:flex">
                  <span>—W</span>
                  <span>—L</span>
                </div>
                <div class="flex flex-col items-end">
                  <span class="text-sm font-semibold text-muted-foreground/40">—</span>
                  <span class="text-xs text-muted-foreground/40">MMR</span>
                </div>
              </div>
            </template>
          </div>
        </template>
      </div>
    </Card>
  </div>
</template>

<script setup lang="ts">
import type { APIGetResponse, APILeaderboardEntry } from '~/types/api'
import { authClient } from '~/lib/client'

definePageMeta({
  layout: 'logged-in',
})

const config = useRuntimeConfig()
const { data: sessionData } = await authClient.getSession()
const token = sessionData?.session.token

const { data: leaderboard, pending } = await useFetch<APIGetResponse<APILeaderboardEntry[]>>(
  `${config.public.apiBasedUrl}/v1/leaderboard`,
  {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    server: false,
  }
)

const entries = computed<(APILeaderboardEntry | null)[]>(() => {
  const data = leaderboard.value?.data ?? []
  return [...data, ...Array(Math.max(0, 10 - data.length)).fill(null)]
})
</script>

<style lang="postcss" scoped></style>
