/**
 * Production build of the renderer as a plain web app.
 *
 * The same sources ship twice: electron.vite.config.ts wraps them in the Electron
 * shell, this config serves them over HTTP. Everything Electron-only sits behind
 * the accessors in src/renderer/src/lib/platform.ts, so no source branching is
 * needed here.
 *
 * Deliberately NOT derived from electron.vite.config.ts: that one carries main
 * and preload targets plus externalizeDeps, none of which apply to a browser
 * bundle. vite.qa.config.mjs is the closer relative -- it already proved the
 * renderer runs as a plain Vite app -- but it hardcodes demo mode and a
 * localhost backend, which must not leak into a production build.
 *
 *   npm run build:web   -> out/web
 *   npm run dev:web     -> :5175, proxying the API to a local backend
 */
import { resolve } from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { vendorChunks } from './build-chunks.mjs'

/** Backend to proxy to during `dev:web`. Not used by the production build. */
const DEV_BACKEND = process.env.WEB_DEV_BACKEND || 'http://localhost:8080'

export default defineConfig({
  root: resolve('src/renderer'),

  // Relative asset paths, so the bundle does not care whether it is served from
  // the domain root or a sub-path.
  base: './',

  resolve: {
    alias: { '@': resolve('src/renderer/src') },
  },

  plugins: [
    // Same compiler settings as the Electron build -- otherwise the web bundle
    // would ship un-memoised components the desktop build optimises.
    react({
      babel: {
        plugins: [['babel-plugin-react-compiler', { compilationMode: 'annotation' }]],
      },
    }),
    tailwindcss(),
  ],

  build: {
    outDir: resolve('out/web'),
    emptyOutDir: true,
    sourcemap: false,
    chunkSizeWarningLimit: 250,
    rollupOptions: {
      output: { manualChunks: vendorChunks },
    },
  },

  server: {
    port: 5175,
    strictPort: true,
    // In production the app and the API share an origin, so the renderer derives
    // the backend address from window.location (lib/constants.ts). This proxy
    // reproduces that locally instead of setting RENDERER_VITE_API_URL, so dev
    // exercises the same code path production uses.
    proxy: {
      '/api': { target: DEV_BACKEND, changeOrigin: true, ws: true },
      '/guest': { target: DEV_BACKEND, changeOrigin: true },
      '/webhooks': { target: DEV_BACKEND, changeOrigin: true },
      '/reset-password': { target: DEV_BACKEND, changeOrigin: true },
    },
  },
})
