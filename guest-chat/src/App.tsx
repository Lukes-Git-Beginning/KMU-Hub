import { useCallback, useEffect, useState } from 'react'
import { apiClient } from './api/client'
import type { GuestChannelConfig, GuestSession } from './api/types'
import { ChatWindow } from './components/ChatWindow'
import { PreChatForm } from './components/PreChatForm'

type AppState =
  | { phase: 'loading' }
  | { phase: 'prechat'; token: string; config: GuestChannelConfig }
  | { phase: 'chat'; token: string; session: GuestSession; config: GuestChannelConfig }
  | { phase: 'expired' }
  | { phase: 'error'; message: string }

function extractToken(): string | null {
  const path = window.location.pathname
  // Path format: /guest/:token
  const match = path.match(/\/guest\/([^/]+)/)
  return match ? match[1] : null
}

export function App() {
  const [state, setState] = useState<AppState>({ phase: 'loading' })

  useEffect(() => {
    const token = extractToken()
    if (!token) {
      setState({ phase: 'error', message: 'Kein Chat-Token gefunden.' })
      return
    }

    apiClient.validateToken(token).then(
      (result) => {
        if (!result.session.is_active) {
          setState({ phase: 'expired' })
          return
        }
        // Apply branding
        if (result.config?.primary_color) {
          document.documentElement.style.setProperty('--primary', result.config.primary_color)
        }
        setState({ phase: 'prechat', token, config: result.config })
      },
      () => {
        setState({ phase: 'expired' })
      },
    )
  }, [])

  const handleSessionCreated = useCallback(
    (token: string, session: GuestSession, config: GuestChannelConfig) => {
      apiClient.setToken(token)
      setState({ phase: 'chat', token, session, config })
    },
    [],
  )

  switch (state.phase) {
    case 'loading':
      return (
        <div className="screen-center">
          <div className="spinner" />
        </div>
      )

    case 'prechat':
      return (
        <PreChatForm
          token={state.token}
          config={state.config}
          onSessionCreated={handleSessionCreated}
        />
      )

    case 'chat':
      return (
        <ChatWindow
          token={state.token}
          session={state.session}
          config={state.config}
        />
      )

    case 'expired':
      return (
        <div className="screen-center">
          <div className="error-icon">&#128337;</div>
          <div className="error-title">Chat abgelaufen</div>
          <div className="error-message">
            Dieser Chat-Link ist nicht mehr gueltig.<br />
            Bitte kontaktieren Sie uns fuer einen neuen Link.
          </div>
        </div>
      )

    case 'error':
      return (
        <div className="screen-center">
          <div className="error-icon">&#9888;&#65039;</div>
          <div className="error-title">Fehler</div>
          <div className="error-message">{state.message}</div>
        </div>
      )
  }
}
