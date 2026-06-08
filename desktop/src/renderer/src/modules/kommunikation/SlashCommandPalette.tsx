/**
 * Slash-command palette (mock-shell, Phase 5).
 *
 * Opens above the reply composer when the input starts with `/`. Lists the
 * available commands filtered by the typed query. There is no bot/command
 * backend yet (see backend-gaps.md) — selecting a command inserts a mock
 * placeholder and the host shows a hint toast. The command registry +
 * keyboard nav are real so wiring a backend later is a drop-in.
 */
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Image, BarChart3, AlarmClock, Sparkles } from 'lucide-react'

export interface SlashCommand {
  name: string
  labelKey: string
  descKey: string
  icon: typeof Image
}

export const SLASH_COMMANDS: SlashCommand[] = [
  { name: 'giphy', labelKey: 'kommunikation.slash.giphy', descKey: 'kommunikation.slash.giphyDesc', icon: Image },
  { name: 'umfrage', labelKey: 'kommunikation.slash.poll', descKey: 'kommunikation.slash.pollDesc', icon: BarChart3 },
  { name: 'erinnerung', labelKey: 'kommunikation.slash.reminder', descKey: 'kommunikation.slash.reminderDesc', icon: AlarmClock },
]

interface SlashCommandPaletteProps {
  query: string
  onSelect: (command: SlashCommand) => void
  onClose: () => void
}

export function SlashCommandPalette({ query, onSelect, onClose }: SlashCommandPaletteProps) {
  const { t } = useTranslation()
  const [selectedIndex, setSelectedIndex] = useState(0)
  const listRef = useRef<HTMLDivElement>(null)

  const filtered = query
    ? SLASH_COMMANDS.filter((c) => c.name.startsWith(query.toLowerCase()))
    : SLASH_COMMANDS

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- reset on results change
    setSelectedIndex(0)
  }, [filtered.length])

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
        const cmd = filtered[selectedIndex]
        if (cmd) onSelect(cmd)
      } else if (e.key === 'Escape') {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [filtered, selectedIndex, onSelect, onClose])

  if (filtered.length === 0) return null

  return (
    <div className="absolute bottom-full left-0 z-50 mb-1 w-72 rounded-lg border border-border bg-card py-1 shadow-lg">
      <p className="mb-1 flex items-center gap-1.5 px-3 pt-1 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
        <Sparkles className="h-3 w-3" />
        {t('kommunikation.slash.title')}
      </p>
      <div ref={listRef} className="max-h-48 overflow-y-auto">
        {filtered.map((cmd, i) => {
          const Icon = cmd.icon
          return (
            <button
              key={cmd.name}
              onClick={() => onSelect(cmd)}
              className={`flex w-full items-center gap-2.5 px-3 py-1.5 text-left transition-colors ${
                i === selectedIndex ? 'bg-secondary' : 'hover:bg-secondary/50'
              }`}
            >
              <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
                <Icon className="h-3.5 w-3.5" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-foreground">/{cmd.name}</p>
                <p className="truncate text-[11px] text-muted-foreground">{t(cmd.descKey)}</p>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
