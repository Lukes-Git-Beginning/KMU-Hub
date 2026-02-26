import { type FormEvent, useState } from 'react'
import { apiClient } from '../api/client'
import type { GuestChannelConfig, GuestSession } from '../api/types'

interface PreChatFormProps {
  token: string
  config: GuestChannelConfig
  onSessionCreated: (token: string, session: GuestSession, config: GuestChannelConfig) => void
}

export function PreChatForm({ token, config, onSessionCreated }: PreChatFormProps) {
  const [displayName, setDisplayName] = useState('')
  const [email, setEmail] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    const name = displayName.trim()
    if (!name) {
      setError('Bitte geben Sie Ihren Namen ein.')
      return
    }
    if (name.length > 200) {
      setError('Name darf maximal 200 Zeichen lang sein.')
      return
    }
    const emailVal = email.trim() || undefined
    if (emailVal && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(emailVal)) {
      setError('Bitte geben Sie eine gueltige E-Mail-Adresse ein.')
      return
    }

    setIsSubmitting(true)
    setError(null)

    try {
      // We need the channel_id from the validated session -- re-validate to get it
      const validation = await apiClient.validateToken(token)
      const result = await apiClient.createSession(
        validation.session.channel_id,
        name,
        emailVal,
      )
      onSessionCreated(result.token, result.session, config)
    } catch {
      setError('Chat konnte nicht gestartet werden. Bitte versuchen Sie es erneut.')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="prechat-container">
      <div className="prechat-card">
        {config.logo_url && (
          <img src={config.logo_url} alt="" className="prechat-logo" />
        )}
        <h2>Willkommen</h2>
        <p className="prechat-welcome">{config.welcome_message}</p>
        <form onSubmit={handleSubmit}>
          <div className="form-field">
            <label htmlFor="displayName">Name *</label>
            <input
              id="displayName"
              type="text"
              placeholder="Ihr Name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              maxLength={200}
              disabled={isSubmitting}
              autoFocus
            />
          </div>
          <div className="form-field">
            <label htmlFor="email">E-Mail (optional)</label>
            <input
              id="email"
              type="email"
              placeholder="E-Mail (optional)"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={isSubmitting}
            />
          </div>
          {error && <div className="form-error">{error}</div>}
          <button type="submit" className="btn-primary" disabled={isSubmitting}>
            {isSubmitting ? 'Wird gestartet...' : 'Chat starten'}
          </button>
        </form>
      </div>
    </div>
  )
}
