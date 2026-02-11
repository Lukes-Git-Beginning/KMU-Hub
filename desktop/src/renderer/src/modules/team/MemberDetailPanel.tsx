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
} from 'lucide-react'
import { DetailPanel } from '@/components/shared'
import type { TeamMember } from '@/stores/team'

const statusLabels: Record<string, string> = {
  online: 'Online',
  away: 'Abwesend',
  offline: 'Offline',
  dnd: 'Nicht stoeren',
}

const statusColors: Record<string, string> = {
  online: 'bg-success',
  away: 'bg-warning',
  offline: 'bg-text-disabled',
  dnd: 'bg-error',
}

interface MemberDetailPanelProps {
  member: TeamMember
  onClose: () => void
  onEmail: () => void
  onCall: () => void
  onMessage: () => void
  onEdit: () => void
}

export function MemberDetailPanel({
  member,
  onClose,
  onEmail,
  onCall,
  onMessage,
  onEdit,
}: MemberDetailPanelProps) {
  const fullName = `${member.firstName} ${member.lastName}`
  const tenure = getTenure(member.joinDate)

  return (
    <DetailPanel title="Mitglied-Details" onClose={onClose}>
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
              {member.initials}
            </div>
            <span
              className={`absolute bottom-0 right-0 h-4 w-4 rounded-full border-2 border-card ${statusColors[member.status]}`}
            />
          </div>
        </div>
      </div>

      <div className="mt-6 space-y-4">
        {/* Name & Role */}
        <div>
          <h3 className="text-base font-medium text-foreground">{fullName}</h3>
          <p className="text-xs text-muted-foreground">{member.role}</p>
          <div className="flex items-center gap-2 mt-1">
            <span className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${
              member.isActive ? 'bg-success-light text-success' : 'bg-error-light text-error'
            }`}>
              <span className={`h-1.5 w-1.5 rounded-full ${member.isActive ? 'bg-success' : 'bg-error'}`} />
              {member.isActive ? statusLabels[member.status] : 'Deaktiviert'}
            </span>
            <span className="text-[10px] text-muted-foreground">
              {member.contractType} · {member.workload}%
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
        <section className="space-y-2">
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Kontakt</h4>
          <div className="space-y-1.5 text-xs">
            <div className="flex items-center gap-2 text-muted-foreground">
              <Mail className="h-3.5 w-3.5 shrink-0" />
              <a href={`mailto:${member.email}`} className="text-primary hover:underline">{member.email}</a>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <Phone className="h-3.5 w-3.5 shrink-0" />
              <span>{member.phone}</span>
            </div>
            {member.mobile && (
              <div className="flex items-center gap-2 text-muted-foreground">
                <Phone className="h-3.5 w-3.5 shrink-0" />
                <span>{member.mobile} (Mobil)</span>
              </div>
            )}
            <div className="flex items-center gap-2 text-muted-foreground">
              <MapPin className="h-3.5 w-3.5 shrink-0" />
              <span>{member.location}</span>
            </div>
          </div>
        </section>

        {/* Employment */}
        <section className="space-y-2">
          <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Anstellung</h4>
          <div className="space-y-1.5 text-xs">
            <div className="flex items-center gap-2 text-muted-foreground">
              <Briefcase className="h-3.5 w-3.5 shrink-0" />
              <span>{member.department}</span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <Shield className="h-3.5 w-3.5 shrink-0" />
              <span>{member.contractType} ({member.workload}%)</span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <Calendar className="h-3.5 w-3.5 shrink-0" />
              <span>Seit {new Date(member.joinDate).toLocaleDateString('de-CH')} ({tenure})</span>
            </div>
            {member.manager && (
              <div className="flex items-center gap-2 text-muted-foreground">
                <Briefcase className="h-3.5 w-3.5 shrink-0" />
                <span>Vorgesetzt: {member.manager}</span>
              </div>
            )}
          </div>
        </section>

        {/* Current Task */}
        {member.currentTask && (
          <section className="space-y-2">
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Aktuelle Aufgabe</h4>
            <div className="flex items-center gap-2 text-xs text-foreground rounded-md bg-secondary/50 px-3 py-2">
              <Clock className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
              <span>{member.currentTask}</span>
            </div>
          </section>
        )}

        {/* Projects */}
        {member.projects.length > 0 && (
          <section className="space-y-2">
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Projekte</h4>
            <div className="flex flex-wrap gap-1.5">
              {member.projects.map((p) => (
                <span key={p} className="rounded-full bg-primary/10 px-2.5 py-0.5 text-[10px] font-medium text-primary">
                  {p}
                </span>
              ))}
            </div>
          </section>
        )}

        {/* Skills */}
        {member.skills.length > 0 && (
          <section className="space-y-2">
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Skills</h4>
            <div className="flex flex-wrap gap-1.5">
              {member.skills.map((s) => (
                <span key={s} className="rounded-full bg-secondary px-2.5 py-0.5 text-[10px] text-muted-foreground">
                  {s}
                </span>
              ))}
            </div>
          </section>
        )}

        {/* Notes */}
        {member.notes && (
          <section className="space-y-2">
            <h4 className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Notizen</h4>
            <p className="text-xs text-muted-foreground leading-relaxed">{member.notes}</p>
          </section>
        )}

        {/* Edit button */}
        <button
          onClick={onEdit}
          className="w-full rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
        >
          Profil bearbeiten
        </button>
      </div>
    </DetailPanel>
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
