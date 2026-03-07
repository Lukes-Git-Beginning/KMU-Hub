/**
 * DuplicateDetectionDialog — Checks for duplicate contacts and offers merge.
 *
 * Side-by-side comparison with field-level merge selection.
 * Mock matching on email, phone, and name similarity.
 */
import { useState, useMemo } from 'react'
import {
  AlertTriangle,
  GitMerge,
  Search,
} from 'lucide-react'
import { toast } from 'sonner'
import type { Contact } from '@/stores/contacts'
import { useContacts } from '@/api/hooks/useContacts'
import { backendContactToUI } from './adapters'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { DuplicateMatchCard } from './DuplicateMatchCard'
import { MergeFieldSelector } from './MergeFieldSelector'

// ---------------------------------------------------------------------------
// Duplicate matching logic (mock)
// ---------------------------------------------------------------------------

interface DuplicateMatch {
  contact: Contact
  score: number
  reasons: string[]
}

function findDuplicates(target: Contact, contacts: Contact[]): DuplicateMatch[] {
  const matches: DuplicateMatch[] = []

  for (const c of contacts) {
    if (c.id === target.id) continue
    let score = 0
    const reasons: string[] = []

    // Email match (highest weight)
    if (target.email && c.email && target.email.toLowerCase() === c.email.toLowerCase()) {
      score += 50
      reasons.push('Gleiche E-Mail')
    }

    // Phone match
    const normalizePhone = (p: string) => p.replace(/\s+/g, '').replace(/[^0-9+]/g, '')
    if (target.phone && c.phone && normalizePhone(target.phone) === normalizePhone(c.phone)) {
      score += 30
      reasons.push('Gleiche Telefonnummer')
    }

    // Name similarity
    const nameA = `${target.firstName} ${target.lastName}`.toLowerCase()
    const nameB = `${c.firstName} ${c.lastName}`.toLowerCase()
    if (nameA === nameB) {
      score += 40
      reasons.push('Gleicher Name')
    } else if (target.lastName.toLowerCase() === c.lastName.toLowerCase()) {
      score += 20
      reasons.push('Gleicher Nachname')
    }

    // Company match
    if (target.company && c.company && target.company.toLowerCase() === c.company.toLowerCase()) {
      score += 10
      reasons.push('Gleiche Firma')
    }

    if (score >= 20) {
      matches.push({ contact: c, score: Math.min(score, 100), reasons })
    }
  }

  return matches.sort((a, b) => b.score - a.score)
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface DuplicateDetectionDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  targetContact: Contact | null
}

type MergeSelections = Record<string, 'a' | 'b'>

export function DuplicateDetectionDialog({
  open,
  onOpenChange,
  targetContact,
}: DuplicateDetectionDialogProps) {
  const { data } = useContacts()
  const contacts = (data?.contacts ?? []).map(backendContactToUI)
  const [selectedMatchId, setSelectedMatchId] = useState<string | null>(null)
  const [mergeSelections, setMergeSelections] = useState<MergeSelections>({})
  const [step, setStep] = useState<'list' | 'merge'>('list')

  const duplicates = useMemo(() => {
    if (!targetContact) return []
    return findDuplicates(targetContact, contacts)
  }, [targetContact, contacts])

  const selectedMatch = duplicates.find((d) => d.contact.id === selectedMatchId)

  const handleSelectMatch = (matchId: string) => {
    setSelectedMatchId(matchId)
    // Initialize merge selections with 'a' (keep original)
    setMergeSelections({
      firstName: 'a',
      lastName: 'a',
      email: 'a',
      phone: 'a',
      mobile: 'a',
      company: 'a',
      jobTitle: 'a',
      department: 'a',
      website: 'a',
      notes: 'a',
    })
    setStep('merge')
  }

  const handleMerge = () => {
    toast.success('Kontakte zusammengefuehrt')
    onOpenChange(false)
    setStep('list')
    setSelectedMatchId(null)
  }

  const handleBack = () => {
    setStep('list')
    setSelectedMatchId(null)
  }

  if (!targetContact) return null

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) { setStep('list'); setSelectedMatchId(null) }; onOpenChange(o) }}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {step === 'list' ? (
              <>
                <AlertTriangle className="h-5 w-5 text-warning" />
                Duplikaterkennung
              </>
            ) : (
              <>
                <GitMerge className="h-5 w-5 text-primary" />
                Kontakte zusammenfuehren
              </>
            )}
          </DialogTitle>
          <DialogDescription>
            {step === 'list'
              ? `Moegliche Duplikate fuer "${targetContact.firstName} ${targetContact.lastName}".`
              : 'Waehle fuer jedes Feld den Wert den du behalten moechtest.'}
          </DialogDescription>
        </DialogHeader>

        {step === 'list' && (
          <div className="space-y-3 pt-2">
            {duplicates.length === 0 ? (
              <div className="py-8 text-center">
                <Search className="mx-auto h-10 w-10 text-muted-foreground/30" />
                <p className="mt-3 text-sm font-medium text-foreground">Keine Duplikate gefunden</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  Dieser Kontakt scheint einzigartig zu sein.
                </p>
              </div>
            ) : (
              <>
                <p className="text-xs text-muted-foreground">
                  {duplicates.length} moegliche{duplicates.length === 1 ? 's' : ''} Duplikat{duplicates.length !== 1 ? 'e' : ''} gefunden.
                  Waehle einen Eintrag zum Zusammenfuehren.
                </p>
                <div className="space-y-2">
                  {duplicates.map((dup) => (
                    <DuplicateMatchCard
                      key={dup.contact.id}
                      contact={dup.contact}
                      matchScore={dup.score}
                      matchReasons={dup.reasons}
                      isSelected={selectedMatchId === dup.contact.id}
                      onSelect={() => handleSelectMatch(dup.contact.id)}
                    />
                  ))}
                </div>
              </>
            )}

            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => onOpenChange(false)}
                className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
              >
                Schliessen
              </button>
            </div>
          </div>
        )}

        {step === 'merge' && selectedMatch && (
          <div className="space-y-4 pt-2">
            {/* Column headers */}
            <div className="grid grid-cols-[100px_1fr_1fr] gap-2 items-center">
              <span />
              <div className="rounded-md bg-primary/10 px-2 py-1 text-center text-xs font-medium text-primary">
                Original (A)
              </div>
              <div className="rounded-md bg-secondary px-2 py-1 text-center text-xs font-medium text-muted-foreground">
                Duplikat (B)
              </div>
            </div>

            {/* Name row - info only */}
            <div className="grid grid-cols-[100px_1fr_1fr] gap-2 items-center">
              <span className="text-[11px] text-muted-foreground text-right">Name</span>
              <span className="text-sm font-medium text-foreground">
                {targetContact.firstName} {targetContact.lastName}
              </span>
              <span className="text-sm font-medium text-foreground">
                {selectedMatch.contact.firstName} {selectedMatch.contact.lastName}
              </span>
            </div>

            {/* Field selectors */}
            <div className="space-y-1.5">
              <MergeFieldSelector
                label="E-Mail"
                valueA={targetContact.email}
                valueB={selectedMatch.contact.email}
                selected={mergeSelections.email ?? 'a'}
                onSelect={(s) => setMergeSelections((prev) => ({ ...prev, email: s }))}
              />
              <MergeFieldSelector
                label="Telefon"
                valueA={targetContact.phone}
                valueB={selectedMatch.contact.phone}
                selected={mergeSelections.phone ?? 'a'}
                onSelect={(s) => setMergeSelections((prev) => ({ ...prev, phone: s }))}
              />
              <MergeFieldSelector
                label="Mobil"
                valueA={targetContact.mobile}
                valueB={selectedMatch.contact.mobile}
                selected={mergeSelections.mobile ?? 'a'}
                onSelect={(s) => setMergeSelections((prev) => ({ ...prev, mobile: s }))}
              />
              <MergeFieldSelector
                label="Firma"
                valueA={targetContact.company}
                valueB={selectedMatch.contact.company}
                selected={mergeSelections.company ?? 'a'}
                onSelect={(s) => setMergeSelections((prev) => ({ ...prev, company: s }))}
              />
              <MergeFieldSelector
                label="Position"
                valueA={targetContact.jobTitle}
                valueB={selectedMatch.contact.jobTitle}
                selected={mergeSelections.jobTitle ?? 'a'}
                onSelect={(s) => setMergeSelections((prev) => ({ ...prev, jobTitle: s }))}
              />
              <MergeFieldSelector
                label="Abteilung"
                valueA={targetContact.department}
                valueB={selectedMatch.contact.department}
                selected={mergeSelections.department ?? 'a'}
                onSelect={(s) => setMergeSelections((prev) => ({ ...prev, department: s }))}
              />
              <MergeFieldSelector
                label="Website"
                valueA={targetContact.website}
                valueB={selectedMatch.contact.website}
                selected={mergeSelections.website ?? 'a'}
                onSelect={(s) => setMergeSelections((prev) => ({ ...prev, website: s }))}
              />
              <MergeFieldSelector
                label="Notizen"
                valueA={targetContact.notes}
                valueB={selectedMatch.contact.notes}
                selected={mergeSelections.notes ?? 'a'}
                onSelect={(s) => setMergeSelections((prev) => ({ ...prev, notes: s }))}
              />
            </div>

            {/* Actions */}
            <div className="flex justify-between pt-2">
              <button
                onClick={handleBack}
                className="h-9 rounded-md border border-border px-4 text-sm text-foreground hover:bg-accent transition-colors"
              >
                Zurueck
              </button>
              <button
                onClick={handleMerge}
                className="flex items-center gap-1.5 h-9 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
              >
                <GitMerge className="h-3.5 w-3.5" />
                Zusammenfuehren
              </button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
