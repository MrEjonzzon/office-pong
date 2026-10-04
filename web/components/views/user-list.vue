<template>
  <Card>
    <CardHeader class="pb-3">
      <CardTitle>Players</CardTitle>
    </CardHeader>

    <div class="divide-y divide-border border-t border-border">
      <!-- Skeleton -->
      <template v-if="!challenges">
        <div v-for="i in 5" :key="i" class="flex animate-pulse items-center gap-4 px-4 py-3">
          <div class="size-8 shrink-0 rounded-full bg-muted" />
          <div class="flex-1">
            <div class="h-4 w-32 rounded bg-muted" />
          </div>
          <div class="h-9 w-14 rounded-md bg-muted" />
        </div>
      </template>

      <!-- Unified player list -->
      <template v-else-if="challenges">
        <div
          v-for="player in allPlayers"
          :key="player.id"
          class="flex items-center gap-4 px-4 py-3">
          <NuxtImg v-if="player.image" :src="player.image" class="size-8 shrink-0 rounded-full" />
          <div v-else class="size-8 shrink-0 rounded-full bg-muted" />

          <span class="flex-1 truncate font-medium">{{ player.name }}</span>
          <span
            v-if="player.challenge && player.status !== 'none'"
            class="shrink-0 text-xs text-muted-foreground">
            {{ formatLabel(player.challenge.bestOf) }}
          </span>

          <div class="ml-auto flex shrink-0 gap-2">
            <!-- incoming: Accept + Deny -->
            <template v-if="player.status === 'incoming'">
              <Button
                class="h-9"
                variant="default"
                :disabled="loadingIds[player.id]"
                @click="acceptChallenge(player)">
                Accept
              </Button>
              <Button
                class="h-9"
                variant="destructive"
                :disabled="loadingIds[player.id]"
                @click="declineChallenge(player)">
                Deny
              </Button>
            </template>

            <!-- outgoing pending: waiting -->
            <Button
              v-else-if="player.status === 'outgoing'"
              class="h-9 w-14 p-3"
              variant="secondary"
              disabled>
              <Icon name="lucide:clock-fading" class="size-full" />
            </Button>

            <!-- active game: go to game -->
            <Button
              v-else-if="player.status === 'active'"
              class="h-9 w-14 bg-green-500 p-3"
              variant="default"
              @click="router.push(`/game/${player.challenge?.gameId}`)">
              <Icon name="lucide:play" class="size-full" />
            </Button>

            <!-- no challenge, picking a format: sets picker -->
            <template v-else-if="pickingFor === player.id">
              <Button
                v-for="n in bestOfOptions"
                :key="n"
                class="h-9 w-9 p-0"
                variant="destructive"
                :disabled="loadingIds[player.id]"
                @click="createChallenge(player, n)">
                {{ n }}
              </Button>
              <Button class="h-9 w-9 p-2" variant="ghost" @click="pickingFor = null">
                <Icon name="lucide:x" class="size-full" />
              </Button>
            </template>

            <!-- no challenge: challenge button -->
            <Button
              v-else
              class="h-9 w-14 p-3"
              variant="destructive"
              :disabled="loadingIds[player.id]"
              @click="pickingFor = player.id">
              <Icon
                v-if="loadingIds[player.id]"
                name="lucide:loader-circle"
                class="size-full animate-spin" />
              <Icon v-else name="lucide:swords" class="size-full" />
            </Button>
          </div>
        </div>

        <div v-if="!allPlayers.length" class="px-4 py-8 text-center text-sm text-muted-foreground">
          No other players yet.
        </div>
      </template>
    </div>
  </Card>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, reactive } from 'vue'
import { useWindowFocus } from '@vueuse/core'
import type { APIChallenge, APIGetResponse, APIUser } from '~/types/api'
import { authClient } from '~/lib/client'
import { toast } from '~/components/ui/toast/use-toast'

const router = useRouter()
const { data: session } = await useSession(useFetch)
const { data: sessionData } = await authClient.getSession()
const config = useRuntimeConfig()
const token = sessionData?.session.token

const loadingIds = reactive<Record<string, boolean>>({})

/* Match format: the challenger picks the number of sets */
const bestOfOptions = [1, 3, 5, 7]
const pickingFor = ref<string | null>(null)
const formatLabel = (bestOf?: number) => (!bestOf || bestOf === 1 ? '1 set' : `Bo${bestOf}`)

/* Polling */
const focused = useWindowFocus()
let pollInterval: number | undefined

watch(focused, isFocused => (isFocused ? poll() : stopPoll()))
onMounted(() => poll())
onUnmounted(() => stopPoll())

const poll = () => {
  pollInterval = window.setInterval(() => {
    executeGetChallenges()
    executeUsers()
  }, 2000)
}
const stopPoll = () => {
  if (pollInterval) clearInterval(pollInterval)
}

/* Fetches */
const { data: challenges, execute: executeGetChallenges } = await useFetch<
  APIGetResponse<APIChallenge[]>
>(`${config.public.apiBasedUrl}/v1/user/${session.value?.user.id}/challenges`, {
  headers: { Authorization: `Bearer ${token}` },
  server: false,
})

const { data: users, execute: executeUsers } = await useFetch<APIGetResponse<APIUser[]>>(
  `${config.public.apiBasedUrl}/v1/users`,
  { headers: { Authorization: `Bearer ${token}` } }
)

/* Navigate challenger to game when challenge is accepted */
const navigatedGameIds = new Set<string>()
watch(
  () => challenges.value?.data,
  data => {
    if (!data) return
    for (const c of data) {
      if (
        c.challenger === session.value?.user.id &&
        c.state === 'active' &&
        c.gameId &&
        !navigatedGameIds.has(c.gameId)
      ) {
        navigatedGameIds.add(c.gameId)
        router.push(`/game/${c.gameId}`)
        return
      }
    }
  }
)

/* Unified player list */
type PlayerRow = {
  id: string
  name: string
  image: string
  status: 'none' | 'incoming' | 'outgoing' | 'active'
  challenge?: APIChallenge
}

const allPlayers = computed<PlayerRow[]>(() => {
  if (!users.value?.data) return []

  const myId = session.value?.user.id

  return users.value.data
    .filter(u => u.id !== myId)
    .map(u => {
      const challenge = challenges.value?.data?.find(
        c =>
          (c.challenger === u.id || c.challengee === u.id) &&
          (c.state === 'pending' || c.state === 'active')
      )

      if (!challenge) return { ...u, status: 'none' as const }
      if (challenge.state === 'active') return { ...u, status: 'active' as const, challenge }
      if (challenge.challenger === myId) return { ...u, status: 'outgoing' as const, challenge }
      return { ...u, status: 'incoming' as const, challenge }
    })
})

/* Actions */
const createChallenge = async (player: PlayerRow, bestOf: number) => {
  pickingFor.value = null
  loadingIds[player.id] = true
  const err = await $fetch(
    `${config.public.apiBasedUrl}/v1/user/${session.value?.user.id}/challenge/${player.id}`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: { bestOf },
    }
  )
    .then(() => null)
    .catch(e => e)
  if (err) {
    toast({ title: 'Failed to send challenge', variant: 'destructive' })
  } else {
    toast({ title: `Challenge sent to ${player.name} (${formatLabel(bestOf)})` })
  }
  await executeGetChallenges()
  loadingIds[player.id] = false
}

const acceptChallenge = async (player: PlayerRow) => {
  if (!player.challenge) return
  loadingIds[player.id] = true
  const res = await $fetch<APIGetResponse<string>>(
    `${config.public.apiBasedUrl}/v1/challenge/${player.challenge.id}/accept`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    }
  ).catch(e => e)
  loadingIds[player.id] = false
  if (res instanceof Error) {
    toast({ title: 'Failed to accept challenge', variant: 'destructive' })
    return
  }
  if (res?.data) router.push(`/game/${res.data}`)
}

const declineChallenge = async (player: PlayerRow) => {
  if (!player.challenge) return
  loadingIds[player.id] = true
  const err = await $fetch(
    `${config.public.apiBasedUrl}/v1/challenge/${player.challenge.id}/decline`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    }
  )
    .then(() => null)
    .catch(e => e)
  if (err) {
    toast({ title: 'Failed to decline challenge', variant: 'destructive' })
  }
  await executeGetChallenges()
  loadingIds[player.id] = false
}
</script>

<style lang="postcss" scoped></style>
