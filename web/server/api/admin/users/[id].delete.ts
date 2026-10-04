import { pool } from '~/utils/auth'

// Deletes a user and everything tied to them (same order as scripts/delete-user.sh).
// Other players' ratings are not recalculated.
export default defineEventHandler(async event => {
  const id = getRouterParam(event, 'id')
  if (!id) throw createError({ statusCode: 400, message: 'Missing user id' })

  const client = await pool.connect()
  let deleted = 0
  try {
    await client.query('BEGIN')
    await client.query(
      `DELETE FROM "games" WHERE "winnerId" = $1 OR "challengeId" IN
         (SELECT "id" FROM "challenges" WHERE "challenger" = $1 OR "challengee" = $1 OR "createdBy" = $1)`,
      [id]
    )
    await client.query(
      `DELETE FROM "challenges" WHERE "challenger" = $1 OR "challengee" = $1 OR "createdBy" = $1`,
      [id]
    )
    await client.query(`DELETE FROM "user_stats" WHERE "userId" = $1`, [id])
    await client.query(`DELETE FROM "session" WHERE "userId" = $1`, [id])
    await client.query(`DELETE FROM "account" WHERE "userId" = $1`, [id])
    deleted = (await client.query(`DELETE FROM "user" WHERE "id" = $1`, [id])).rowCount ?? 0
    await client.query('COMMIT')
  } catch (e) {
    await client.query('ROLLBACK')
    throw e
  } finally {
    client.release()
  }

  if (!deleted) throw createError({ statusCode: 404, message: 'User not found' })
  return { ok: true }
})
