// QA-only Vite server for the parallel-batch sub-terminal. Serves the renderer
// as a plain web app on :5174 with demo-mode forced on, so Playwright
// screenshot-QA runs without colliding with the main terminal (:5173) or
// touching the tracked electron.vite.config.ts. Not for production.
import { resolve } from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  root: resolve('src/renderer'),
  define: {
    'import.meta.env.RENDERER_VITE_DEMO_MODE': '"true"',
    'import.meta.env.RENDERER_VITE_API_URL': '"http://localhost:8080"',
  },
  resolve: {
    alias: { '@': resolve('src/renderer/src') },
  },
  plugins: [react(), tailwindcss()],
  server: { port: 5174, strictPort: true },
})
