import { betterAuth } from 'better-auth'
import { username } from 'better-auth/plugins'

import { Pool } from 'pg'

export const auth = betterAuth({
  database: new Pool({
    connectionString: (() => {
      const config = useRuntimeConfig()
      return `postgres://${config.dbUser}:${config.dbPassword}@${config.dbHost}:${config.dbPort}/${config.dbName}`
    })(),
    ssl: (() => {
      const config = useRuntimeConfig()
      return config.dbSsl === 'false' ||
        config.dbHost === 'localhost' ||
        config.dbHost === '127.0.0.1'
        ? false
        : { rejectUnauthorized: false }
    })(),
  }),
  emailAndPassword: {
    enabled: true,
  },
  // Login is username + password. better-auth still requires an email column, so the
  // client stores a synthetic, never-used `<username>@users.pingis.invalid` address.
  plugins: [username()],
  // Google creds in .env belong to a previous employer's project — do not use them.
  // TODO: set up our own Google OAuth client, then re-enable socialProviders.google.
})
