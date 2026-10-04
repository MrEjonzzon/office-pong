import { pool } from '~/utils/auth'

export default defineEventHandler(async () => {
  const { rows } = await pool.query(`
    SELECT u."id", u."username", u."name", u."createdAt",
           COALESCE(s."mmr", 1500) AS "mmr", COALESCE(s."wins", 0) AS "wins", COALESCE(s."losses", 0) AS "losses"
    FROM "user" u
    LEFT JOIN "user_stats" s ON s."userId" = u."id"
    ORDER BY u."createdAt"
  `)
  return rows
})
