interface TypingIndicatorProps {
  user: string | null
}

export function TypingIndicator({ user }: TypingIndicatorProps) {
  if (!user) return <div className="typing-indicator" />

  return (
    <div className="typing-indicator">
      <span>{user} schreibt</span>
      <div className="typing-dots">
        <span />
        <span />
        <span />
      </div>
    </div>
  )
}
