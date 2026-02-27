/**
 * Header mini-widget — total unread message count.
 */
import { MessageSquare } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

// Mock unread count — in production this would come from chat store
const MOCK_UNREAD = 7

export function UnreadCountWidget() {
  const navigate = useNavigate()

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          onClick={() => navigate('/chat')}
          className="relative flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs hover:bg-accent transition-colors"
        >
          <MessageSquare className="h-3.5 w-3.5 text-muted-foreground" />
          {MOCK_UNREAD > 0 && (
            <span className="flex h-4 min-w-[16px] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
              {MOCK_UNREAD > 99 ? '99+' : MOCK_UNREAD}
            </span>
          )}
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        {MOCK_UNREAD > 0
          ? `${MOCK_UNREAD} ungelesene Nachrichten`
          : 'Keine ungelesenen Nachrichten'}
      </TooltipContent>
    </Tooltip>
  )
}
