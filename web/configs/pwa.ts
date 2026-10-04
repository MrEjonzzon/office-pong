import type { ModuleOptions } from '@vite-pwa/nuxt/dist/module.js'

export const pwaConfig = {
  workbox: {
    navigateFallback: '/',
  },
  devOptions: {
    enabled: true,
    type: 'module',
  },
  selfDestroying: true,
  manifest: {
    name: 'Office Pong',
    short_name: 'Office Pong',
    icons: [
      {
        src: 'manifest-icon-192.maskable.png',
        sizes: '192x192',
        type: 'image/png',
        purpose: 'any',
      },
      {
        src: 'manifest-icon-192.maskable.png',
        sizes: '192x192',
        type: 'image/png',
        purpose: 'maskable',
      },
      {
        src: 'manifest-icon-512.maskable.png',
        sizes: '512x512',
        type: 'image/png',
        purpose: 'any',
      },
      {
        src: 'manifest-icon-512.maskable.png',
        sizes: '512x512',
        type: 'image/png',
        purpose: 'maskable',
      },
    ],
    start_url: '/',
    display: 'fullscreen',
    background_color: '#FFFFFF',
    theme_color: '#FFFFFF',
  },
} satisfies Partial<ModuleOptions> | undefined
