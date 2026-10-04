import { betterAuth } from 'better-auth'
import { username } from 'better-auth/plugins'

import { Pool } from 'pg'

// Shared with the admin API routes (server/api/admin)
export const pool = new Pool({
  connectionString: (() => {
    const config = useRuntimeConfig()
    return `postgres://${config.dbUser}:${config.dbPassword}@${config.dbHost}:${config.dbPort}/${config.dbName}`
  })(),
  ssl: (() => {
    const config = useRuntimeConfig()
    // Nuxt parses NUXT_DB_SSL=false into a boolean, so compare as a string
    return String(config.dbSsl) === 'false' ||
      config.dbHost === 'localhost' ||
      config.dbHost === '127.0.0.1'
      ? false
      : { rejectUnauthorized: false }
  })(),
})

export const auth = betterAuth({
  database: pool,
  emailAndPassword: {
    enabled: true,
  },
  // Login is username + password. better-auth still requires an email column, so the
  // client stores a synthetic, never-used `<username>@users.pingis.invalid` address.
  plugins: [username()],
  // Google creds in .env belong to a previous employer's project — do not use them.
  // TODO: set up our own Google OAuth client, then re-enable socialProviders.google.
})
