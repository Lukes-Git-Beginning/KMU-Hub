/**
 * Team Chat widget — recent messages from team channels.
 */
import { memo } from 'react'
import { Hash, ArrowRight } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import type { WidgetProps } from '@/components/widgets/WidgetRegistry'

interface MockMessage {
  id: string
  channel: string
  sender: string
  avatar: string
  text: string
  time: string
  unread: boolean
}

const MOCK_MESSAGES: MockMessage[] = [
  { id: '1', channel: 'allgemein', sender: 'Anna S.', avatar: 'AS', text: 'Hey Team, kurze Erinnerung: Sprint Review heute um 14 Uhr!', time: '08:45', unread: true },
  { id: '2', channel: 'entwicklung', sender: 'Markus W.', avatar: 'MW', text: 'PR #247 ist ready for review, kann jemand drueberschauen?', time: '08:32', unread: true },
  { id: '3', channel: 'allgemein', sender: 'Lena F.', avatar: 'LF', text: 'Die neuen Designs fuer das Dashboard sind im Figma. Link im Thread.', time: '08:15', unread: false },
  { id: '4', channel: 'vertrieb', sender: 'Thomas M.', avatar: 'TM', text: 'Meier GmbH hat den Vertrag unterschrieben! 🎉', time: 'Gestern', unread: false },
  { id: '5', channel: 'random', sender: 'Sarah B.', avatar: 'SB', text: 'Wer hat Lust auf Mittagessen beim Italiener?', time: 'Gestern', unread: false },
]

function TeamChat(_props: WidgetProps) {
  const navigate = useNavigate()
  const unreadCount = MOCK_MESSAGES.filter((m) => m.unread).length

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between px-4 pt-4 pb-2">
        <span className="text-xs text-muted-foreground">
          {unreadCount > 0 && (
            <span className="inline-flex h-4 min-w-[16px] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white mr-1.5">
              {unreadCount}
            </span>
          )}
          Letzte Nachrichten
        </span>
        <button
          onClick={() => navigate('/chat')}
          className="flex items-center gap-0.5 text-[10px] text-primary hover:underline"
        >
          Alle anzeigen <ArrowRight className="h-2.5 w-2.5" />
        </button>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-auto divide-y divide-border">
        {MOCK_MESSAGES.map((msg) => (
          <div
            key={msg.id}
            className={`flex items-start gap-3 px-4 py-2.5 cursor-pointer transition-colors hover:bg-accent/50 ${
              msg.unread ? 'bg-primary/5' : ''
            }`}
            onClick={() => navigate('/chat')}
          >
            <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary/10 text-[10px] font-semibold text-primary mt-0.5">
              {msg.avatar}
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5">
                <span className="text-xs font-semibold text-foreground">{msg.sender}</span>
                <span className="flex items-center gap-0.5 text-[10px] text-muted-foreground">
                  <Hash className="h-2.5 w-2.5" />{msg.channel}
                </span>
                <span className="text-[10px] text-muted-foreground ml-auto shrink-0">{msg.time}</span>
              </div>
              <p className="text-xs text-muted-foreground truncate mt-0.5">{msg.text}</p>
            </div>
            {msg.unread && (
              <span className="mt-2 h-2 w-2 shrink-0 rounded-full bg-primary" />
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

export default memo(TeamChat)
