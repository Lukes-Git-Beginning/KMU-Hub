// Demo mode MUST be imported first — it overrides globalThis.fetch at
// module scope so that openapi-fetch (and every other client) captures
// the intercepted version.
import { startDemoMode } from './mocks/demo-mode'

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { onlineManager } from '@tanstack/react-query'
import './styles/globals.css'
import App from './App'
import { useAuthStore } from './stores/auth'

// Clear stale query cache in demo mode.
startDemoMode()

// Configure TanStack Query's online manager to use browser online/offline events.
// When offline, TanStack Query automatically pauses queries and mutations.
// When online, it resumes and refetches stale data.
onlineManager.setEventListener((setOnline) => {
  const onlineHandler = () => setOnline(true)
  const offlineHandler = () => setOnline(false)
  window.addEventListener('online', onlineHandler)
  window.addEventListener('offline', offlineHandler)
  // Set initial state
  setOnline(navigator.onLine)
  return () => {
    window.removeEventListener('online', onlineHandler)
    window.removeEventListener('offline', offlineHandler)
  }
})

// Initialize auth state (check for stored tokens) before rendering.
// This runs asynchronously -- the App component shows a loading state
// while isLoading is true in the auth store.
useAuthStore.getState().initialize()

const root = document.getElementById('root')
if (root) {
  createRoot(root).render(
    <StrictMode>
      <App />
    </StrictMode>
  )
}
