import { useState } from 'react'
import {
  Mail,
  Phone,
  MessageSquare,
  Briefcase,
  MapPin,
  Calendar,
  Clock,
  Shield,
  PhoneCall,
  Loader2,
  FileText,
  Upload,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { DetailPanel } from '@/components/shared'
import { toast } from 'sonner'
import {
  useEmployee,
  useEmployeeLeaveBalance,
  useEmployeeDocuments,
  useDocumentCategories,
  useUploadEmployeeDocument,
} from '@/api/hooks/hr-hooks'
import type { EmployeeProfile } from '@/api/hr-types'

const contractTypeLabels: Record<string, string> = {
  full_time: 'Vollzeit',
  part_time: 'Teilzeit',
  praktikum: 'Praktikum',
  freelance: 'Freelancer',
}

interface MemberDetailPanelProps {
  memberId: string
  memberName?: string
  memberInitials?: string
  onClose: () => void
  onEmail: () => void
  onCall: () => void
  onMessage: () => void
  onEdit: () => void
}

export function MemberDetailPanel({
  memberId,
  memberName,
  memberInitials,
  onClose,
  onEmail,
  onCall,
  onMessage,
  onEdit,
}: MemberDetailPanelProps) {
  const { data: employee, isLoading } = useEmployee(memberId)
  const { data: balance } = useEmployeeLeaveBalance(employee?.userId ?? '')
  const { data: documents } = useEmployeeDocuments(memberId)

  const fullName = employee
    ? (employee.userName ?? `${employee.department ?? 'Mitarbeiter'}`)
    : (memberName ?? 'Laden...')

  const initials = memberInitials ?? fullName
    .split(' ')
    .map((n) => n[0])
    .join('')
    .slice(0, 2)
    .toUpperCase()

  const tenure = employee ? getTenure(employee.startDate) : ''

  return (
    <DetailPanel open title="Mitglied-Details" onClose={onClose}>
      {/* Header with gradient */}
      <div className="relative -mx-5 -mt-1 mb-4">
        <div
          className="h-20 rounded-t-lg"
          style={{
            background: `linear-gradient(135deg, var(--primary) 0%, color-mix(in srgb, var(--primary) 60%, #000) 100%)`,
          }}
        />
        <div className="absolute -bottom-8 left-5 flex items-end gap-3">
          <div className="relative">
            <div className="flex h-16 w-16 items-center justify-center rounded-full border-4 border-card bg-primary-light text-lg font-bold text-primary">
              {initials}
            </div>
          </div>
        </div>
      </div>

      {isLoading ? (
        <div className="mt-6 flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : (
        <div className="mt-6 space-y-4">
          {/* Name & Role */}
          <div>
            <h3 className="text-base font-medium text-foreground">{fullName}</h3>
            <p className="text-xs text-muted-foreground">{employee?.positionTitle ?? ''}</p>
            <div className="flex items-center gap-2 mt-1">
              <span className="text-[10px] text-muted-foreground">
                {contractTypeLabels[employee?.contractType ?? ''] ?? employee?.contractType}
                {employee?.workDaysPerWeek ? ` · ${employee.workDaysPerWeek} Tage/Woche` : ''}
              </span>
            </div>
          </div>

          {/* Quick Actions */}
          <div className="flex gap-2">
            <button
              onClick={onEmail}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <Mail className="h-3.5 w-3.5" />
              E-Mail
            </button>
            <button
              onClick={onCall}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <PhoneCall className="h-3.5 w-3.5" />
              Anrufen
            </button>
            <button
              onClick={onMessage}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <MessageSquare className="h-3.5 w-3.5" />
              Chat
            </button>
          </div>

          {/* Contact Info */}
          {employee?.userEmail && (
            <section className="space-y-2">
              <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Kontakt</h4>
              <div className="space-y-1.5 text-xs">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Mail className="h-3.5 w-3.5 shrink-0" />
                  <a href={`mailto:${employee.userEmail}`} className="text-primary hover:underline">{employee.userEmail}</a>
                </div>
                {employee.emergencyContactPhone && (
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <Phone className="h-3.5 w-3.5 shrink-0" />
                    <span>{employee.emergencyContactPhone} (Notfallkontakt: {employee.emergencyContactName})</span>
                  </div>
                )}
                {employee.addressCity && (
                  <div className="flex items-center gap-2 text-muted-foreground">
                    <MapPin className="h-3.5 w-3.5 shrink-0" />
                    <span>
                      {[employee.addressStreet, employee.addressPostalCode, employee.addressCity].filter(Boolean).join(', ')}
                    </span>
                  </div>
                )}
              </div>
            </section>
          )}

          {/* Employment */}
          <section className="space-y-2">
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Anstellung</h4>
            <div className="space-y-1.5 text-xs">
              {employee?.department && (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Briefcase className="h-3.5 w-3.5 shrink-0" />
                  <span>{employee.department}</span>
                </div>
              )}
              <div className="flex items-center gap-2 text-muted-foreground">
                <Shield className="h-3.5 w-3.5 shrink-0" />
                <span>{contractTypeLabels[employee?.contractType ?? ''] ?? employee?.contractType}</span>
              </div>
              {employee?.startDate && (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Calendar className="h-3.5 w-3.5 shrink-0" />
                  <span>Seit {new Date(employee.startDate).toLocaleDateString('de-DE')} ({tenure})</span>
                </div>
              )}
              {employee?.managerName && (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Briefcase className="h-3.5 w-3.5 shrink-0" />
                  <span>Vorgesetzt: {employee.managerName}</span>
                </div>
              )}
            </div>
          </section>

          {/* Leave Balance */}
          {balance && (
            <section className="space-y-2">
              <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Urlaubsanspruch</h4>
              <div className="rounded-lg border border-border bg-secondary/30 p-3">
                <div className="flex items-end gap-2 mb-2">
                  <span className="text-2xl font-bold text-primary">{balance.remaining}</span>
                  <span className="text-sm text-muted-foreground mb-0.5">/ {balance.totalEntitlement} Tage</span>
                </div>
                <div className="w-full h-2 rounded-full bg-secondary overflow-hidden mb-2">
                  <div
                    className="h-full rounded-full bg-primary transition-all"
                    style={{ width: `${Math.min(100, Math.round((balance.used / balance.totalEntitlement) * 100))}%` }}
                  />
                </div>
                <div className="flex justify-between text-[10px] text-muted-foreground">
                  <span>{balance.used} genommen</span>
                  {balance.carriedOver > 0 && (
                    <span className={balance.carryoverExpired ? 'text-error' : ''}>
                      {balance.carriedOver} Uebertrag{balance.carryoverExpired ? ' (abgelaufen)' : ''}
                    </span>
                  )}
                </div>
              </div>
            </section>
          )}

          {/* Documents */}
          <DocumentsSection memberId={memberId} documents={documents ?? []} />

          {/* Edit button */}
          <button
            onClick={onEdit}
            className="w-full rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
          >
            Profil bearbeiten
          </button>
        </div>
      )}
    </DetailPanel>
  )
}

// ============================================================
// Documents Section with upload
// ============================================================
function DocumentsSection({
  memberId,
  documents,
}: {
  memberId: string
  documents: { id: string; fileName?: string; categoryName?: string; createdAt: string }[]
}) {
  const { data: categories } = useDocumentCategories(memberId)
  const uploadMutation = useUploadEmployeeDocument()

  const [expanded, setExpanded] = useState(false)
  const [showUpload, setShowUpload] = useState(false)
  const [categoryId, setCategoryId] = useState('')
  const [notes, setNotes] = useState('')
  const [fileName, setFileName] = useState('')

  const handleUpload = () => {
    if (!categoryId) {
      toast.error('Bitte Kategorie waehlen')
      return
    }
    // In real app, fileId comes from a file upload service. Here we simulate.
    const mockFileId = `file-${Date.now()}`
    uploadMutation.mutate(
      {
        employeeId: memberId,
        data: {
          categoryId,
          fileId: mockFileId,
          notes: notes.trim() || undefined,
        },
      },
      {
        onSuccess: () => {
          setShowUpload(false)
          setCategoryId('')
          setNotes('')
          setFileName('')
        },
      },
    )
  }

  const Chevron = expanded ? ChevronUp : ChevronDown

  return (
    <section className="space-y-2">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1 w-full text-left"
      >
        <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
          Dokumente ({documents.length})
        </h4>
        <Chevron className="h-3 w-3 text-muted-foreground ml-auto" />
      </button>

      {expanded && (
        <div className="space-y-2">
          {/* Document list */}
          {documents.length > 0 ? (
            <div className="space-y-1">
              {documents.map((doc) => (
                <div key={doc.id} className="flex items-center gap-2 text-xs text-muted-foreground">
                  <FileText className="h-3.5 w-3.5 shrink-0" />
                  <span className="truncate">{doc.fileName ?? doc.categoryName ?? 'Dokument'}</span>
                  <span className="text-[10px] ml-auto shrink-0">
                    {new Date(doc.createdAt).toLocaleDateString('de-DE')}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground italic">Keine Dokumente vorhanden</p>
          )}

          {/* Upload area */}
          {!showUpload ? (
            <Button
              variant="outline"
              size="sm"
              className="w-full text-xs"
              onClick={() => setShowUpload(true)}
            >
              <Upload className="mr-1.5 h-3.5 w-3.5" />
              Dokument hochladen
            </Button>
          ) : (
            <div className="rounded-lg border border-border bg-secondary/30 p-3 space-y-2">
              <div className="space-y-1">
                <Label className="text-xs">Kategorie</Label>
                <select
                  value={categoryId}
                  onChange={(e) => setCategoryId(e.target.value)}
                  className="w-full rounded border border-border bg-input-background px-2 py-1.5 text-xs text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
                >
                  <option value="">Waehlen...</option>
                  {(categories ?? []).map((cat) => (
                    <option key={cat.id} value={cat.id}>{cat.name}</option>
                  ))}
                  {/* Fallback categories if API not loaded */}
                  {!categories && (
                    <>
                      <option value="contract">Vertrag</option>
                      <option value="certificate">Zeugnis</option>
                      <option value="id">Ausweis</option>
                      <option value="cert">Zertifikat</option>
                      <option value="other">Sonstiges</option>
                    </>
                  )}
                </select>
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Datei</Label>
                <Input
                  type="file"
                  onChange={(e) => setFileName(e.target.files?.[0]?.name ?? '')}
                  className="text-xs h-8"
                />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">Notizen (optional)</Label>
                <Input
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  placeholder="z.B. Gueltigkeit, Bemerkungen"
                  className="text-xs h-8"
                />
              </div>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  className="flex-1 text-xs h-7"
                  onClick={handleUpload}
                  disabled={uploadMutation.isPending}
                >
                  {uploadMutation.isPending ? (
                    <Loader2 className="mr-1 h-3 w-3 animate-spin" />
                  ) : (
                    <Upload className="mr-1 h-3 w-3" />
                  )}
                  Hochladen
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="text-xs h-7"
                  onClick={() => { setShowUpload(false); setCategoryId(''); setNotes(''); setFileName('') }}
                >
                  Abbrechen
                </Button>
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  )
}

function getTenure(joinDate: string): string {
  const join = new Date(joinDate)
  const now = new Date()
  const months = (now.getFullYear() - join.getFullYear()) * 12 + (now.getMonth() - join.getMonth())
  if (months < 1) return 'Neu'
  if (months < 12) return `${months} Monat${months > 1 ? 'e' : ''}`
  const years = Math.floor(months / 12)
  const rem = months % 12
  return rem > 0 ? `${years}J ${rem}M` : `${years} Jahr${years > 1 ? 'e' : ''}`
}
