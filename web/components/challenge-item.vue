<template>
  <div class="flex items-center justify-between gap-4">
    <div v-if="user" class="flex min-w-0 shrink items-center gap-4">
      <NuxtImg v-if="user.image" :src="user.image" class="size-8 rounded-full" />
      <div class="flex flex-col overflow-hidden">
        <span class="line-clamp-1">{{ user.name }}</span>
        <span class="line-clamp-1 text-xs text-muted-foreground">{{ user.id }}</span>
      </div>
    </div>
    <div v-if="type === 'incoming'" class="flex justify-end gap-3">
      <Button class="h-10 shrink-0" variant="default" @click="acceptChallenge()"> Accept </Button>
      <Button class="h-10 shrink-0" variant="destructive" @click="declineChallenge()">
        Deny
      </Button>
    </div>

    <Button
      v-else-if="type === 'outgoing'"
      disabled
      class="ml-auto h-10 w-14 p-3"
      variant="secondary">
      <Icon name="lucide:clock-fading" class="size-full" />
    </Button>
  </div>
</template>

<script setup lang="ts">
import type { APIChallenge, APIGetResponse } from '~/types/api'
import type { ListUser } from '~/types/users'
const router = useRouter()

const props = defineProps<{
  type: 'incoming' | 'outgoing'
  user?: ListUser
  challenge?: APIChallenge
}>()

const { data: sessionData } = await authClient.getSession()
const token = sessionData?.session.token

const config = useRuntimeConfig()

const acceptChallenge = async () => {
  if (!props.challenge) return
  const res = await $fetch<APIGetResponse<string>>(
    `${config.public.apiBasedUrl}/v1/challenge/${props.challenge?.id}/accept`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      method: 'POST',
      onResponseError: error => {
        console.error('Error accepting challenge:', error)
      },
    }
  )
  router.push(`/game/${res.data}`)
}

const declineChallenge = async () => {
  if (!props.challenge) return
  await $fetch(`${config.public.apiBasedUrl}/v1/challenge/${props.challenge?.id}/decline`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    method: 'POST',
    onResponse: ({ response: _ }) => {
      // Handle successful response
    },
  })
}
</script>
