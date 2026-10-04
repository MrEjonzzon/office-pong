import { createHash, timingSafeEqual } from 'node:crypto'

// HTTP Basic auth for /admin and /api/admin/*, using NUXT_ADMIN_USERNAME / NUXT_ADMIN_PASSWORD.
// Fails closed: with no admin password configured the admin area is disabled.
const digest = (s: string) => createHash('sha256').update(s).digest()
const same = (a: string, b: string) => timingSafeEqual(digest(a), digest(b))

export default defineEventHandler(event => {
  const path = getRequestURL(event).pathname
  const isApi = path === '/api/admin' || path.startsWith('/api/admin/')
  if (!isApi && path !== '/admin' && !path.startsWith('/admin/')) return

  const config = useRuntimeConfig(event)
  // String(): Nuxt parses env overrides, so a numeric-looking password could arrive as a number
  const expectedUser = String(config.adminUsername || 'admin')
  const expectedPass = String(config.adminPassword || '')
  if (!expectedPass) throw createError({ statusCode: 503, statusMessage: 'Admin disabled' })

  const [scheme, encoded] = (getHeader(event, 'authorization') || '').split(' ')
  let ok = false
  if (scheme?.toLowerCase() === 'basic' && encoded) {
    const decoded = Buffer.from(encoded, 'base64').toString()
    const i = decoded.indexOf(':')
    if (i >= 0) {
      const userOk = same(decoded.slice(0, i), expectedUser)
      const passOk = same(decoded.slice(i + 1), expectedPass)
      ok = userOk && passOk
    }
  }
  if (!ok) {
    setResponseHeader(event, 'WWW-Authenticate', 'Basic realm="admin"')
    throw createError({ statusCode: 401, statusMessage: 'Unauthorized' })
  }

  // Browsers attach cached Basic credentials to cross-site requests too, so require a custom
  // header on writes (a cross-site page can't send one without a CORS preflight, which we never allow).
  if (isApi && event.method !== 'GET' && getHeader(event, 'x-requested-with') !== 'admin') {
    throw createError({ statusCode: 403, statusMessage: 'Forbidden' })
  }
})
