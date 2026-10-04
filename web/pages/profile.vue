<template>
  <div v-if="session">
    <div class="flex flex-col items-center justify-center space-y-4">
      <NuxtImg v-if="session.user.image" :src="session.user.image" class="h-16 w-16 rounded-full" />
      <h1 class="text-2xl font-bold">{{ session.user.name }}</h1>
      <!-- TODO: prettify -->
      <div class="flex items-center flex-col">
        <p class="py-0">MecRating : {{ stats?.mmr ?? '...' }}</p>
        <p class="py-0">Wins : {{ stats?.wins ?? '...' }}</p>
        <p class="py-0">Losses : {{ stats?.losses ?? '...' }}</p>
      </div>

      <Button class="rounded-full px-6 py-2" @click="handleSignOut">Sign Out</Button>

      <Button class="rounded-full px-6 py-2" @click="toggleSession">
        {{ showSession ? 'Hide Session Data' : 'Show Session Data' }}
      </Button>

      <pre
        v-if="showSession"
        class="w-full max-w-80 overflow-x-auto whitespace-pre-wrap break-words rounded-lg bg-gray-100 p-4 text-xs dark:bg-gray-800">
        {{ session }}
        </pre
      >
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { authClient } from '~/lib/client'
import { useRouter } from 'vue-router'
import type { APIUserStats } from '~/types/api'
import { toast } from '~/components/ui/toast/use-toast'

definePageMeta({
  layout: 'logged-in',
})

const { data: session } = await useSession(useFetch)
const router = useRouter()
const showSession = ref(false)
const config = useRuntimeConfig()
const { data: sessionData } = await authClient.getSession()
const token = sessionData?.session.token
const loading = ref(false)

function toggleSession() {
  showSession.value = !showSession.value
}

async function handleSignOut() {
  const { error } = await authClient.signOut({
    fetchOptions: {
      onSuccess: () => router.push('/login'),
    },
  })
  if (error) toast({ title: 'Sign out failed', description: error.message, variant: 'destructive' })
}

const stats = ref()
onMounted(async () => {
  stats.value = await getUserStats(sessionData?.user.id)
})

const getUserStats = async (userId?: string): Promise<APIUserStats | undefined> => {
  if (!userId) return
  loading.value = true
  const res = await $fetch<APIUserStats>(`${config.public.apiBasedUrl}/v1/user/${userId}/stats`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    method: 'GET',
  })
  loading.value = false
  return res
}
</script>

<style lang="postcss" scoped></style>
