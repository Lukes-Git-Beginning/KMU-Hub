/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly RENDERER_VITE_API_URL?: string
  readonly RENDERER_VITE_DEMO_MODE?: string
  readonly RENDERER_VITE_BOOKING_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
