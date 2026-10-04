import { pool } from '~/utils/auth'

// Changes the display name only; the login username stays the same.
export default defineEventHandler(async event => {
  const id = getRouterParam(event, 'id')
  const body = await readBody<{ name?: string }>(event)
  const name = String(body?.name ?? '').trim()

  if (!id) throw createError({ statusCode: 400, message: 'Missing user id' })
  if (name.length < 1 || name.length > 50) {
    throw createError({ statusCode: 400, message: 'Name must be 1-50 characters' })
  }

  const res = await pool.query(`UPDATE "user" SET "name" = $1, "updatedAt" = NOW() WHERE "id" = $2`, [name, id])
  if (!res.rowCount) throw createError({ statusCode: 404, message: 'User not found' })
  return { ok: true }
})
