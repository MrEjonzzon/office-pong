import { authClient } from '~/lib/client'

export default defineNuxtRouteMiddleware(async to => {
  // /admin has its own HTTP Basic auth (server/middleware/admin-auth.ts)
  if (to.path.startsWith('/admin')) return

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
