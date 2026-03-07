/**
 * DuplicateMatchCard — Shows a potential duplicate match with similarity score.
 */
import { AlertTriangle, Mail, Phone, Building2 } from 'lucide-react'
import type { Contact } from '@/stores/contacts'

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface DuplicateMatchCardProps {
  contact: Contact
  matchScore: number // 0-100
  matchReasons: string[]
  isSelected: boolean
  onSelect: () => void
}

export function DuplicateMatchCard({
  contact,
  matchScore,
  matchReasons,
  isSelected,
  onSelect,
}: DuplicateMatchCardProps) {
  const scoreColor =
    matchScore >= 80
      ? 'text-error bg-error-light'
      : matchScore >= 50
        ? 'text-warning bg-warning-light'
        : 'text-muted-foreground bg-secondary'

  return (
    <button
      onClick={onSelect}
      className={`flex w-full flex-col rounded-lg border p-3 text-left transition-colors ${
        isSelected
          ? 'border-primary bg-primary/5'
          : 'border-border hover:bg-accent/50'
      }`}
    >
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-secondary text-xs font-medium">
            {contact.initials}
          </div>
          <div>
            <p className="text-sm font-medium text-foreground">
              {contact.title ? `${contact.title} ` : ''}{contact.firstName} {contact.lastName}
            </p>
            <p className="text-[11px] text-muted-foreground">{contact.jobTitle}</p>
          </div>
        </div>
        <div className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${scoreColor}`}>
          <AlertTriangle className="h-3 w-3" />
          {matchScore}% Match
        </div>
      </div>

      {/* Contact details */}
      <div className="mt-2 space-y-1 pl-10">
        {contact.email && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Mail className="h-3 w-3" />
            {contact.email}
          </div>
        )}
        {contact.phone && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Phone className="h-3 w-3" />
            {contact.phone}
          </div>
        )}
        {contact.company && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Building2 className="h-3 w-3" />
            {contact.company}
          </div>
        )}
      </div>

      {/* Match reasons */}
      <div className="mt-2 flex flex-wrap gap-1 pl-10">
        {matchReasons.map((reason) => (
          <span key={reason} className="rounded-full bg-secondary px-2 py-0.5 text-[9px] text-muted-foreground">
            {reason}
          </span>
        ))}
      </div>
    </button>
  )
}
