<template>
  <div class="h-screen w-full overflow-y-auto">
    <div class="mx-auto flex max-w-3xl flex-col gap-4 p-4">
      <h1 class="text-2xl font-semibold">Admin</h1>

      <Card>
        <CardHeader class="pb-3">
          <CardTitle>Create user</CardTitle>
        </CardHeader>
        <CardContent class="flex flex-wrap gap-2">
          <Input v-model="newUsername" placeholder="username" class="max-w-48" autocomplete="off" />
          <Input
            v-model="newPassword"
            placeholder="password (min 8)"
            class="max-w-48"
            autocomplete="off" />
          <Button :disabled="!newUsername || !newPassword" @click="createUser">Create</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader class="pb-3">
          <CardTitle>Users ({{ users.length }})</CardTitle>
        </CardHeader>
        <div class="divide-y divide-border border-t border-border">
          <div v-for="u in users" :key="u.id" class="flex flex-wrap items-center gap-2 px-4 py-3">
            <div class="min-w-0 flex-1">
              <div class="truncate font-medium">{{ u.name }}</div>
              <div class="truncate text-xs text-muted-foreground">
                {{ u.username }} · {{ u.mmr }} MMR · {{ u.wins }}W {{ u.losses }}L
              </div>
            </div>
            <Button variant="secondary" class="h-8" @click="rename(u)">Rename</Button>
            <Button variant="secondary" class="h-8" @click="setPassword(u)">Set password</Button>
            <Button variant="destructive" class="h-8" @click="remove(u)">Delete</Button>
          </div>
          <div v-if="!users.length" class="px-4 py-6 text-center text-sm text-muted-foreground">
            No users.
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { toast } from '~/components/ui/toast/use-toast'

definePageMeta({ layout: false })

interface AdminUser {
  id: string
  username: string | null
  name: string
  mmr: number
  wins: number
  losses: number
}

// custom header required by the server for writes (CSRF guard, see server/middleware/admin-auth.ts)
const headers = { 'x-requested-with': 'admin' }

const users = ref<AdminUser[]>([])
const newUsername = ref('')
const newPassword = ref('')

const errorMessage = (e: any) => e?.data?.message ?? e?.statusMessage ?? e?.message ?? 'Failed'

async function load() {
  try {
    users.value = await $fetch<AdminUser[]>('/api/admin/users')
  } catch (e) {
    toast({ title: 'Could not load users', description: errorMessage(e), variant: 'destructive' })
  }
}
onMounted(load)

async function run(title: string, action: () => Promise<unknown>) {
  try {
    await action()
    toast({ title })
    await load()
  } catch (e) {
    toast({ title: 'Failed', description: errorMessage(e), variant: 'destructive' })
  }
}

const createUser = () =>
  run('User created', async () => {
    await $fetch('/api/admin/users', {
      method: 'POST',
      headers,
      body: { username: newUsername.value, password: newPassword.value },
    })
    newUsername.value = ''
    newPassword.value = ''
  })

function rename(u: AdminUser) {
  const name = window.prompt(`New display name for ${u.username}`, u.name)
  if (name === null) return
  run('Name updated', () =>
    $fetch(`/api/admin/users/${u.id}`, { method: 'PATCH', headers, body: { name } })
  )
}

function setPassword(u: AdminUser) {
  const password = window.prompt(`New password for ${u.username} (min 8 characters)`)
  if (!password) return
  run('Password updated', () =>
    $fetch(`/api/admin/users/${u.id}/password`, { method: 'POST', headers, body: { password } })
  )
}

function remove(u: AdminUser) {
  if (!window.confirm(`Delete ${u.username} and all their games and challenges?`)) return
  run('User deleted', () => $fetch(`/api/admin/users/${u.id}`, { method: 'DELETE', headers }))
}
</script>
