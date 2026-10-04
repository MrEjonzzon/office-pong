import { betterAuth } from 'better-auth'

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
  // Google creds in .env belong to a previous employer's project — do not use them.
  // TODO: set up our own Google OAuth client, then re-enable socialProviders.google.
})
