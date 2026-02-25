import type { GuestChannelConfig, GuestSession } from '../api/types'
import { useGuestChat } from '../hooks/useGuestChat'
import { ConnectionStatus } from './ConnectionStatus'
import { MessageInput } from './MessageInput'
import { MessageList } from './MessageList'
import { TypingIndicator } from './TypingIndicator'

interface ChatWindowProps {
  token: string
  session: GuestSession
  config: GuestChannelConfig
}

export function ChatWindow({ token, session, config }: ChatWindowProps) {
  const {
    messages,
    isLoading,
    isSending,
    sendMessage,
    loadMore,
    hasMore,
    typingUser,
    connectionState,
  } = useGuestChat(token, session)

  return (
    <div className="chat-app">
      <div className="chat-header">
        {config.logo_url && (
          <img src={config.logo_url} alt="" className="chat-header-logo" />
        )}
        <span className="chat-header-title">Kundenservice</span>
      </div>
      <ConnectionStatus state={connectionState} />
      <MessageList
        messages={messages}
        isLoading={isLoading}
        hasMore={hasMore}
        onLoadMore={loadMore}
        sessionId={session.id}
        welcomeMessage={config.welcome_message}
      />
      <TypingIndicator user={typingUser} />
      <MessageInput
        onSend={sendMessage}
        isSending={isSending}
        disabled={connectionState === 'disconnected'}
        channelId={session.channel_id}
        config={config}
      />
    </div>
  )
}
