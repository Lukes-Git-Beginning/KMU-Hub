/**
 * MeetingNotesPanel -- Real-time note-taking during a meeting with auto-save.
 *
 * Side panel shown during active meetings. Features:
 * - Textarea with auto-resize for free-form notes
 * - Private/shared toggle (Radix Switch)
 * - Auto-save every 30 seconds via debounced mutation
 * - Save on unmount for data safety
 * - Visual "Speichern..." / "Gespeichert" indicator
 */
import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { Save, Lock, Globe, Sparkles } from 'lucide-react'
import { toast } from 'sonner'
import { useAIStore } from '@/stores/ai'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { useMeetingNotes, useSaveMeetingNotes } from '@/api/hooks/useMeetings'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface MeetingNotesPanelProps {
  meetingId: string
}

// ---------------------------------------------------------------------------
// Debounce helper
// ---------------------------------------------------------------------------

function debounce<T extends (...args: unknown[]) => void>(fn: T, delay: number): T & { cancel: () => void } {
  let timer: ReturnType<typeof setTimeout> | null = null
  const debounced = ((...args: unknown[]) => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => fn(...args), delay)
  }) as T & { cancel: () => void }
  debounced.cancel = () => {
    if (timer) clearTimeout(timer)
  }
  return debounced
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function MeetingNotesPanel({ meetingId }: MeetingNotesPanelProps) {
  const { data: existingNotes, isLoading } = useMeetingNotes(meetingId)
  const saveMutation = useSaveMeetingNotes()

  const [content, setContent] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)
  const [summaryLoading, setSummaryLoading] = useState(false)
  const aiMeetingsEnabled = useAIStore((s) => s.isModuleEnabled('meetings'))

  const handleSummarize = () => {
    setSummaryLoading(true)
    setTimeout(() => {
      const summary = `## Zusammenfassung\n\n**Teilnehmer:** 4 Personen\n**Dauer:** ca. 45 Minuten\n\n### Besprochene Themen:\n- Projektfortschritt Q1 wurde positiv bewertet\n- Budget für Q2 muss bis Freitag finalisiert werden\n- Neue Kundenanfrage von Müller GmbH priorisieren\n\n### Action Items:\n- [ ] Budget-Entwurf erstellen (Marco, bis Fr 28.02.)\n- [ ] Angebot für Müller GmbH vorbereiten (Sarah, bis Mi 26.02.)\n- [ ] Nächstes Meeting: Montag 10:00 Uhr\n\n---\n\n`
      setContent(summary + content)
      setSummaryLoading(false)
      useAIStore.getState().addActivityLog({
        module: 'Meetings',
        action: 'Zusammenfassung erstellt',
        inputPreview: content.slice(0, 60) || '(Leere Notizen)',
        outputPreview: '3 Themen, 3 Action Items',
      })
      toast.success('Zusammenfassung eingefuegt')
    }, 2000)
  }
  const [lastSaved, setLastSaved] = useState('')
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved'>('idle')
  const contentRef = useRef(content)
  const isPrivateRef = useRef(isPrivate)
  const initializedRef = useRef(false)

  // Keep refs in sync
  useEffect(() => {
    contentRef.current = content
    isPrivateRef.current = isPrivate
  }, [content, isPrivate])

   
  useEffect(() => {
    if (existingNotes && !initializedRef.current) {
      setContent(existingNotes.content)
      setIsPrivate(existingNotes.is_private)
      setLastSaved(existingNotes.content)
      initializedRef.current = true
    }
  }, [existingNotes])

  // Save function
  const doSave = useCallback(
    (text: string, priv: boolean) => {
      setSaveStatus('saving')
      saveMutation.mutate(
        { meetingId, data: { content: text, is_private: priv } },
        {
          onSuccess: () => {
            setLastSaved(text)
            setSaveStatus('saved')
            // Reset status after 3 seconds
            setTimeout(() => setSaveStatus('idle'), 3000)
          },
          onError: () => {
            setSaveStatus('idle')
          },
        },
      )
    },
    [meetingId, saveMutation],
  )

  // Debounced auto-save (30 seconds)
  const debouncedSave = useMemo(
    () => debounce((...args: unknown[]) => doSave(args[0] as string, args[1] as boolean), 30_000),
    [doSave],
  )

  // Trigger debounced save on content change
  useEffect(() => {
    if (content !== lastSaved && initializedRef.current) {
      debouncedSave(content, isPrivate)
    }
  }, [content, isPrivate, lastSaved, debouncedSave])

  // Save on unmount
  useEffect(() => {
    return () => {
      debouncedSave.cancel()
      // Save if content changed since last save
      if (contentRef.current !== lastSaved) {
        // Fire-and-forget save on unmount
        saveMutation.mutate({
          meetingId,
          data: { content: contentRef.current, is_private: isPrivateRef.current },
        })
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [meetingId])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin h-6 w-6 border-2 border-primary border-t-transparent rounded-full" />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-semibold text-foreground">Notizen</h3>
          {aiMeetingsEnabled && (
            <button
              onClick={handleSummarize}
              disabled={summaryLoading || !content.trim()}
              className="flex items-center gap-1 rounded-md px-2 py-1 text-[11px] text-primary hover:bg-primary-light transition-colors disabled:opacity-40"
              title="KI-Zusammenfassung generieren"
            >
              {summaryLoading ? (
                <span className="h-3 w-3 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              ) : (
                <Sparkles className="h-3 w-3" />
              )}
              Zusammenfassen
            </button>
          )}
        </div>
        <div className="flex items-center gap-3">
          {/* Save Status */}
          <span className="flex items-center gap-1 text-xs text-muted-foreground">
            {saveStatus === 'saving' && (
              <>
                <Save className="h-3 w-3 animate-pulse" />
                Speichern...
              </>
            )}
            {saveStatus === 'saved' && (
              <>
                <Save className="h-3 w-3 text-green-500" />
                Gespeichert
              </>
            )}
          </span>

          {/* Private/Shared Toggle */}
          <div className="flex items-center gap-1.5">
            {isPrivate ? (
              <Lock className="h-3.5 w-3.5 text-muted-foreground" />
            ) : (
              <Globe className="h-3.5 w-3.5 text-muted-foreground" />
            )}
            <Switch
              checked={isPrivate}
              onCheckedChange={setIsPrivate}
              id="notes-private"
            />
            <Label htmlFor="notes-private" className="text-xs cursor-pointer">
              {isPrivate ? 'Privat' : 'Geteilt'}
            </Label>
          </div>
        </div>
      </div>

      {/* Notes Textarea */}
      <div className="flex-1 p-4">
        <textarea
          className="w-full h-full min-h-[200px] resize-none rounded-lg border border-border bg-card p-3 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
          placeholder="Notizen hier eingeben... Zeilen mit '- [ ]' oder 'TODO:' werden als Action Items erkannt."
          value={content}
          onChange={(e) => setContent(e.target.value)}
        />
      </div>

      {/* Manual Save Button */}
      <div className="px-4 py-3 border-t border-border">
        <button
          onClick={() => {
            debouncedSave.cancel()
            doSave(content, isPrivate)
          }}
          disabled={content === lastSaved || saveStatus === 'saving'}
          className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs text-primary hover:bg-primary/10 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Save className="h-3.5 w-3.5" />
          Jetzt speichern
        </button>
      </div>
    </div>
  )
}
