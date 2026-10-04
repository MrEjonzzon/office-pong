import { auth } from '~/utils/auth'

export default defineEventHandler(async event => {
  const body = await readBody<{ username?: string; password?: string }>(event)
  const username = String(body?.username ?? '').trim().toLowerCase()
  const password = String(body?.password ?? '')

  if (!/^[a-z0-9_.]{3,30}$/.test(username)) {
    throw createError({ statusCode: 400, message: 'Username must be 3-30 letters, digits, _ or .' })
  }
  if (password.length < 8) {
    throw createError({ statusCode: 400, message: 'Password must be at least 8 characters' })
  }

  try {
    // same call the sign-up form makes; better-auth requires an email, so use the synthetic one
    await auth.api.signUpEmail({
      body: { email: `${username}@users.pingis.invalid`, name: username, username, password },
    })
  } catch (e: any) {
    throw createError({
      statusCode: Number(e?.statusCode) || 400,
      message: e?.body?.message ?? e?.message ?? 'Could not create user',
    })
  }
  return { ok: true }
})
