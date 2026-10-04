import { createAuthClient } from 'better-auth/vue' // make sure to import from better-auth/vue
import { magicLinkClient } from 'better-auth/client/plugins'

export const authClient = createAuthClient({
  //you can pass client configuration here
  plugins: [magicLinkClient()],
})
