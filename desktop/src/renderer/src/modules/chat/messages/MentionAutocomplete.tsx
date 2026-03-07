/**
 * @Mention autocomplete dropdown for chat message input.
 *
 * Appears when the user types `@` in the message input.
 * Shows filtered team members with avatar, name, and role.
 * Supports keyboard navigation (ArrowUp/Down, Enter, Escape).
 */
import { useState, useEffect, useMemo, useRef } from 'react'
import { useTeamStore } from '@/stores/team'
import { usePresenceStore } from '@/stores/presence'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'

interface MentionAutocompleteProps {
  query: string
  onSelect: (displayName: string) => void
  onClose: () => void
}

const PRESENCE_COLORS: Record<string, string> = {
  online: 'bg-emerald-500',
  away: 'bg-amber-400',
  dnd: 'bg-red-500',
  offline: 'bg-gray-400',
}

export function MentionAutocomplete({ query, onSelect, onClose }: MentionAutocompleteProps) {
  const members = useTeamStore((s) => s.members)
  const presenceMap = usePresenceStore((s) => s.presenceMap)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const listRef = useRef<HTMLDivElement>(null)

  const filtered = useMemo(() => {
    if (!query) return members.filter((m) => m.isActive).slice(0, 8)
    const q = query.toLowerCase()
    return members
      .filter(
        (m) =>
          m.isActive &&
          (m.firstName.toLowerCase().includes(q) ||
            m.lastName.toLowerCase().includes(q) ||
            `${m.firstName} ${m.lastName}`.toLowerCase().includes(q))
      )
      .slice(0, 8)
  }, [members, query])

  // Reset selected index when results change
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- reset state on dependency change
    setSelectedIndex(0)
  }, [filtered.length])

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIndex((i) => Math.min(i + 1, filtered.length - 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIndex((i) => Math.max(i - 1, 0))
      } else if (e.key === 'Enter' && filtered.length > 0) {
        e.preventDefault()
        const member = filtered[selectedIndex]
        if (member) onSelect(member.firstName)
      } else if (e.key === 'Escape') {
        onClose()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [filtered, selectedIndex, onSelect, onClose])

  // Scroll selected item into view
  useEffect(() => {
    if (listRef.current) {
      const item = listRef.current.children[selectedIndex] as HTMLElement | undefined
      item?.scrollIntoView({ block: 'nearest' })
    }
  }, [selectedIndex])

  if (filtered.length === 0) {
    return (
      <div className="absolute bottom-full left-0 z-50 mb-1 w-64 rounded-lg border border-border bg-card p-3 shadow-lg">
        <p className="text-xs text-muted-foreground">Keine Mitarbeiter gefunden</p>
      </div>
    )
  }

  return (
    <div className="absolute bottom-full left-0 z-50 mb-1 w-72 rounded-lg border border-border bg-card py-1 shadow-lg">
      <p className="mb-1 px-3 pt-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
        Mitarbeiter
      </p>
      <div ref={listRef} className="max-h-48 overflow-y-auto">
        {filtered.map((member, i) => {
          const presence = presenceMap[member.id] ?? member.status ?? 'offline'
          return (
            <button
              key={member.id}
              onClick={() => onSelect(member.firstName)}
              className={`flex w-full items-center gap-2.5 px-3 py-1.5 text-sm transition-colors ${
                i === selectedIndex ? 'bg-secondary' : 'hover:bg-secondary/50'
              }`}
            >
              <div className="relative">
                <Avatar className="h-7 w-7">
                  <AvatarFallback className="text-[10px]">{member.initials}</AvatarFallback>
                </Avatar>
                <span
                  className={`absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-card ${PRESENCE_COLORS[presence] ?? 'bg-gray-400'}`}
                />
              </div>
              <div className="min-w-0 flex-1 text-left">
                <p className="truncate text-sm font-medium text-foreground">
                  {member.firstName} {member.lastName}
                </p>
                <p className="truncate text-[11px] text-muted-foreground">{member.role}</p>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
