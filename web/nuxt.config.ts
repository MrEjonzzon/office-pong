import { appConfig } from './configs/app'
import { pwaConfig } from './configs/pwa'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  devtools: { enabled: true },

  runtimeConfig: {
    postgres: '',
    // Database configuration
    dbHost: '',
    dbPort: '',
    dbName: '',
    dbUser: '',
    dbPassword: '',
    dbSsl: '',
    public: {
      // Overridden at runtime by NUXT_PUBLIC_API_BASED_URL (reading process.env here would bake it in at build time)
      apiBasedUrl: '',
      apiBasedUrlWs: '',
    },
  },

  modules: [
    '@nuxt/fonts',
    '@nuxt/icon',
    '@nuxt/image',
    '@nuxt/scripts',
    '@nuxt/eslint',
    '@nuxtjs/tailwindcss',
    'shadcn-nuxt',
    '@vite-pwa/nuxt',
    '@nuxtjs/color-mode',
  ],
  app: appConfig,
  pwa: pwaConfig,
  icon: {
    collections: ['material-symbols', 'radix-icons'],
  },

  image: {
    domains: ['lh3.googleusercontent.com'],
  },

  css: ['~/assets/css/tailwind.css'],
  shadcn: {
    /**
     * Prefix for all the imported component
     */
    prefix: '',
    /**
     * Directory that the component lives in.
     * @default "./components/ui"
     */
    componentDir: './components/ui',
  },
  experimental: {
    viewTransition: true,
  },
})
