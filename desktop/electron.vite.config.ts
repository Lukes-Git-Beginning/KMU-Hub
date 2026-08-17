import { resolve } from 'path'
import { defineConfig } from 'electron-vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { visualizer } from 'rollup-plugin-visualizer'
import { vendorChunks } from './build-chunks.mjs'

export default defineConfig({
  main: {
    build: {
      externalizeDeps: true
    }
  },
  preload: {
    build: {
      externalizeDeps: true
    }
  },
  renderer: {
    root: resolve('src/renderer'),
    // Dev-server port is env-gated so a second clone (Sub-terminal) can run on
    // a non-colliding port without touching the default. `npm run dev` stays on
    // 5173; `VITE_DEV_PORT=5174 npm run dev` serves the Sub-lane on 5174.
    // (electron-vite does not forward a `--port` CLI flag, hence the env hook.)
    server: {
      port: Number(process.env.VITE_DEV_PORT) || 5173,
    },
    build: {
      chunkSizeWarningLimit: 250,
      rollupOptions: {
        input: resolve('src/renderer/index.html'),
        // Shared with the web build (vite.web.config.mts) so the two cannot drift.
        output: {
          manualChunks: vendorChunks,
        },
      }
    },
    resolve: {
      alias: {
        '@': resolve('src/renderer/src')
      }
    },
    plugins: [
      react({
        babel: {
          plugins: [
            ['babel-plugin-react-compiler', { compilationMode: 'annotation' }]
          ]
        }
      }),
      tailwindcss(),
      visualizer({
        filename: 'dist/bundle-report.html',
        gzipSize: true,
        brotliSize: true,
        open: false
      })
    ]
  }
})
