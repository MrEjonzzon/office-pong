<template>
  <div v-if="user" class="flex items-center gap-4">
    <NuxtImg v-if="user.image" :src="user.image" class="size-8 rounded-full" />
    <div class="flex flex-col">
      <span>{{ user.name }}</span>
      <span class="text-xs text-muted-foreground">{{ user.id }}</span>
    </div>
    <!-- Loading -->
    <Button
      v-if="loading"
      class="ml-auto h-10 w-14 shrink-0 p-3"
      variant="outline"
      disabled
      @click="createChallenge(user)">
      <Icon name="lucide:loader-circle" class="size-full animate-spin" />
    </Button>
    <!-- Invited -->
    <template v-else>
      <Button
        v-if="!user.state"
        class="ml-auto h-10 w-14 shrink-0 p-3"
        variant="destructive"
        @click="createChallenge(user)">
        <Icon name="lucide:swords" class="size-full" />
      </Button>

      <!-- Pending -->
      <Button
        v-else-if="user.state === 'pending'"
        disabled
        class="ml-auto h-10 w-14 p-3"
        variant="secondary">
        <Icon name="lucide:clock-fading" class="size-full" />
      </Button>

      <!-- Active -->
      <Button
        v-else-if="user.state === 'active'"
        class="ml-auto h-10 w-14 bg-green-500 p-3"
        variant="default">
        <Icon name="lucide:clock-fading" class="size-full" />
      </Button>

      <!-- Denied -->
      <Button
        v-else-if="user.state === 'denied'"
        disabled
        class="ml-auto h-10 w-14 p-3"
        variant="ghost">
        <Icon name="fluent-emoji-high-contrast:chicken" class="size-full" />
      </Button>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { APIChallenge, APIGetResponse } from '~/types/api'
import type { ListUser } from '~/types/users'
import { authClient } from '~/lib/client'
const { data: sessionData } = await authClient.getSession()

const token = sessionData?.session.token

const config = useRuntimeConfig()

interface APIChallengeResponse {
  data?: APIChallenge
  error?: string
}

// State
const loading = ref(false)

defineProps<{
  user?: ListUser
}>()

const emits = defineEmits<{
  (e: 'update', callback: () => void): void
}>()

const { data: session } = await useSession(useFetch)
const createChallenge = async (userToChallenge: ListUser) => {
  loading.value = true
  await new Promise(resolve => setTimeout(resolve, 1000))
  await $fetch<APIGetResponse<APIChallengeResponse>>(
    `${config.public.apiBasedUrl}/v1/user/${session.value?.user.id}/challenge/${userToChallenge.id}`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
      method: 'POST',
      onResponse: ({ response: _ }) => {
        // if (res.status == 201) {
        emits('update', () => {
          loading.value = false
        })
        // }
      },
    }
  )

  loading.value = false
}
</script>

<style lang="postcss" scoped></style>
