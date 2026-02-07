import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/globals.css'

function App() {
  return (
    <div className="bg-background text-foreground min-h-screen flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-4xl font-bold tracking-tight">
          KMU Hub
        </h1>
        <p className="mt-4 text-lg text-muted-foreground">
          Desktop App wird geladen...
        </p>
      </div>
    </div>
  )
}

const root = document.getElementById('root')
if (root) {
  createRoot(root).render(
    <StrictMode>
      <App />
    </StrictMode>
  )
}
