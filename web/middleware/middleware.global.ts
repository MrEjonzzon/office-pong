import { authClient } from '~/lib/client'

export default defineNuxtRouteMiddleware(async to => {
  const { data: session } = await authClient.useSession(useFetch)

  // If trying to access /login and a session exists, redirect to home.
  if (to.path === '/login' && session.value) {
    return navigateTo('/home')
  }

  // For all routes except /login,
  // if no session exists, redirect to /login
  if (to.path !== '/login' && !session.value) {
    return navigateTo('/login')
  }
})
