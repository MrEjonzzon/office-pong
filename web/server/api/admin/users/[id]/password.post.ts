import { hashPassword } from 'better-auth/crypto'
import { pool } from '~/utils/auth'

export default defineEventHandler(async event => {
  const id = getRouterParam(event, 'id')
  const body = await readBody<{ password?: string }>(event)
  const password = String(body?.password ?? '')

  if (!id) throw createError({ statusCode: 400, message: 'Missing user id' })
  if (password.length < 8) {
    throw createError({ statusCode: 400, message: 'Password must be at least 8 characters' })
  }

  const res = await pool.query(
    `UPDATE "account" SET "password" = $1, "updatedAt" = NOW() WHERE "providerId" = 'credential' AND "userId" = $2`,
    [await hashPassword(password), id]
  )
  if (!res.rowCount) throw createError({ statusCode: 404, message: 'User not found' })

  // log the user out everywhere
  await pool.query(`DELETE FROM "session" WHERE "userId" = $1`, [id])
  return { ok: true }
})
