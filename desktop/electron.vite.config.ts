import { resolve } from 'path'
import { defineConfig } from 'electron-vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { visualizer } from 'rollup-plugin-visualizer'

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
    build: {
      chunkSizeWarningLimit: 250,
      rollupOptions: {
        input: resolve('src/renderer/index.html'),
        output: {
          manualChunks: {
            'vendor-react': ['react', 'react-dom', 'react-router-dom'],
            'vendor-query': [
              '@tanstack/react-query',
              '@tanstack/react-query-persist-client',
              '@tanstack/query-async-storage-persister',
            ],
            'vendor-editor': [
              '@tiptap/react',
              '@tiptap/starter-kit',
              '@tiptap/extension-code-block-lowlight',
              '@tiptap/extension-image',
              '@tiptap/extension-link',
              '@tiptap/extension-placeholder',
              '@tiptap/extension-table',
              '@tiptap/extension-table-cell',
              '@tiptap/extension-table-header',
              '@tiptap/extension-table-row',
              '@tiptap/extension-task-item',
              '@tiptap/extension-task-list',
              '@tiptap/extension-text-align',
              '@tiptap/extension-underline',
              'lowlight',
            ],
            'vendor-video': [
              '@livekit/components-react',
              '@livekit/components-styles',
              'livekit-client',
            ],
            'vendor-workflow': ['@xyflow/react'],
            'vendor-dnd': ['@dnd-kit/core', '@dnd-kit/sortable', '@dnd-kit/utilities'],
            'vendor-ui': ['lucide-react'],
            'vendor-i18n': ['i18next', 'react-i18next', 'i18next-icu'],
            'vendor-dates': ['date-fns'],
            'vendor-state': ['zustand'],
          },
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
