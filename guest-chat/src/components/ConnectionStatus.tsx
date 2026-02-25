import { useEffect, useState } from 'react'
import type { ConnectionState } from '../hooks/useWebSocket'

interface ConnectionStatusProps {
  state: ConnectionState
}

export function ConnectionStatus({ state }: ConnectionStatusProps) {
  const [showSuccess, setShowSuccess] = useState(false)
  const [wasDisconnected, setWasDisconnected] = useState(false)

  useEffect(() => {
    if (state === 'disconnected' || state === 'connecting') {
      setWasDisconnected(true)
      setShowSuccess(false)
    } else if (state === 'connected' && wasDisconnected) {
      setShowSuccess(true)
      const timer = setTimeout(() => {
        setShowSuccess(false)
        setWasDisconnected(false)
      }, 2000)
      return () => clearTimeout(timer)
    }
  }, [state, wasDisconnected])

  if (state === 'connecting') {
    return (
      <div className="connection-banner warning">
        Verbindung wird hergestellt...
      </div>
    )
  }

  if (state === 'disconnected') {
    return (
      <div className="connection-banner error">
        Verbindung unterbrochen
      </div>
    )
  }

  if (showSuccess) {
    return (
      <div className="connection-banner success">
        Verbunden
      </div>
    )
  }

  return null
}
