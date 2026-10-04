<script setup lang="ts">
import Logo from '~/public/original-logo.png'
import { toast } from '~/components/ui/toast/use-toast'

const { data: session } = await useSession(useFetch)
const router = useRouter()

const name = ref('')
const email = ref('')
const password = ref('')

async function handleSignIn() {
  const { error } = await authClient.signIn.email({ email: email.value, password: password.value })
  if (error) {
    toast({ title: 'Sign in failed', description: error.message, variant: 'destructive' })
    return
  }
  router.push('/home')
}

async function handleSignUp() {
  const { error } = await authClient.signUp.email({ name: name.value, email: email.value, password: password.value })
  if (error) {
    toast({ title: 'Sign up failed', description: error.message, variant: 'destructive' })
    return
  }
  router.push('/home')
}
</script>

<template>
  <div class="flex h-full w-full items-center justify-center">
    <div class="flex w-full max-w-sm flex-col items-center px-6">
      <!-- branding -->
      <div class="mb-10 flex flex-col items-center text-center">
        <div class="mb-4 rounded-2xl bg-white p-4 shadow-sm ring-1 ring-gray-200">
          <img :src="Logo" class="h-28 w-auto" />
        </div>

        <h1 class="text-2xl font-semibold tracking-tight">Office Pong</h1>
        <p class="mt-1 text-sm italic text-muted-foreground">Corporate friendly competition</p>
      </div>

      <!-- Email/password auth (Google sign-in disabled until we have our own OAuth client) -->
      <form v-if="!session" class="flex w-full flex-col gap-2" @submit.prevent="handleSignIn">
        <Input v-model="name" type="text" placeholder="name" />
        <Input v-model="email" type="email" placeholder="email" />
        <Input v-model="password" type="password" placeholder="password" />
        <div class="flex gap-2">
          <button
            type="submit"
            class="h-12 flex-1 rounded-lg border border-indigo-400 bg-indigo-500 text-white transition hover:bg-indigo-600 active:border-indigo-900 active:bg-indigo-800 dark:border-indigo-600">
            Sign in
          </button>
          <button
            type="button"
            class="h-12 flex-1 rounded-lg border border-indigo-400 text-indigo-500 transition hover:bg-indigo-50 active:bg-indigo-100 dark:text-indigo-400 dark:hover:bg-indigo-950"
            @click="handleSignUp">
            Sign up
          </button>
        </div>
      </form>

      <!-- Session action -->
      <button
        v-if="session"
        class="text-sm text-muted-foreground hover:underline"
        @click="authClient.signOut()">
        Sign out
      </button>
    </div>
  </div>
</template>
