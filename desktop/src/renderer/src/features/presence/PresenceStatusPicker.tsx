/**
 * Manual presence status override dropdown.
 *
 * Allows the user to set their status to Online, Abwesend, or
 * Nicht stoeren. "Im Anruf" and "Offline" are automatic only and
 * cannot be manually selected. Placed in the header or profile area.
 */
import { cn } from '@/lib/cn'
import { usePresenceStore } from '@/stores/presence'
import { useSetPresenceStatus } from '@/api/hooks/usePresence'
import type { PresenceLevel } from '@/api/video-types'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

/** Manually selectable statuses (in_call and offline are automatic only). */
const selectableStatuses: Array<{
  value: PresenceLevel
  label: string
  color: string
}> = [
  { value: 'online', label: 'Online', color: 'bg-green-500' },
  { value: 'away', label: 'Abwesend', color: 'bg-yellow-500' },
  { value: 'dnd', label: 'Nicht stoeren', color: 'bg-red-500' },
]

/** Color mapping for display (includes automatic statuses). */
const statusColors: Record<PresenceLevel, string> = {
  online: 'bg-green-500',
  away: 'bg-yellow-500',
  dnd: 'bg-red-500',
  in_call: 'bg-purple-500',
  offline: 'bg-gray-400',
}

/** Display labels for current status (includes automatic statuses). */
const statusLabels: Record<PresenceLevel, string> = {
  online: 'Online',
  away: 'Abwesend',
  dnd: 'Nicht stoeren',
  in_call: 'Im Anruf',
  offline: 'Offline',
}

export function PresenceStatusPicker() {
  const myStatus = usePresenceStore((s) => s.myStatus)
  const setMyStatus = usePresenceStore((s) => s.setMyStatus)
  const setPresenceStatus = useSetPresenceStatus()

  const handleSelect = (status: PresenceLevel) => {
    setMyStatus(status)
    setPresenceStatus.mutate(status)
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          className="flex items-center gap-2 rounded-md px-2 py-1 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
          aria-label={`Status: ${statusLabels[myStatus]}`}
        >
          <span
            className={cn(
              'inline-block h-2.5 w-2.5 shrink-0 rounded-full',
              statusColors[myStatus],
            )}
          />
          <span className="hidden sm:inline text-xs">{statusLabels[myStatus]}</span>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        {selectableStatuses.map((opt) => (
          <DropdownMenuItem
            key={opt.value}
            onClick={() => handleSelect(opt.value)}
            className={cn(
              'flex items-center gap-2 text-sm',
              myStatus === opt.value && 'bg-accent',
            )}
          >
            <span
              className={cn(
                'inline-block h-2.5 w-2.5 shrink-0 rounded-full',
                opt.color,
              )}
            />
            {opt.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
