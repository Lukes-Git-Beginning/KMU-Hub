import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/globals.css'
import App from './App'
import { useAuthStore } from './stores/auth'

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
