/**
 * CallHistoryDetailModal — Detailansicht eines Anruf-History-Eintrags.
 *
 * Öffnet sich beim Klick auf eine Verlaufs-Zeile (Cosmi-Fenster-Standard via
 * shared/DetailModal). Zeigt alle Gesprächs-Metadaten, Teilnehmer, Notiz und —
 * falls vorhanden — die Aufzeichnung. Footer bietet einen echten
 * Protokoll-Download (Blob) und einen Rückruf-Button, der die Meeting-Form
 * mit vorbefülltem Kontaktnamen öffnet.
 */
import { useTranslation } from 'react-i18next'
import {
  Video,
  Phone,
  PhoneIncoming,
  PhoneOutgoing,
  PhoneMissed,
  Building2,
  PhoneCall,
  Users,
  FileText,
  Download,
  Clock,
  CalendarDays,
} from 'lucide-react'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { formatDate } from '@/lib/format'
import type { CallHistoryEntry } from '../../stores/meetings'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatDuration(minutes: number, t: (k: string) => string): string {
  if (minutes === 0) return t('video.history.detail.noDuration')
  const h = Math.floor(minutes / 60)
  const m = minutes % 60
  return h > 0
    ? `${h} ${t('video.history.detail.hoursShort')} ${m} ${t('video.history.detail.minutesShort')}`
    : `${m} ${t('video.history.detail.minutesShort')}`
}

/** Recording length (stored in seconds) → "mm:ss". */
function formatRecordingLength(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

function directionMeta(direction: CallHistoryEntry['direction']) {
  switch (direction) {
    case 'incoming':
      return { icon: PhoneIncoming, tone: 'text-green-600 dark:text-green-400', bg: 'bg-green-500/15', labelKey: 'video.direction.incoming' }
    case 'outgoing':
      return { icon: PhoneOutgoing, tone: 'text-blue-600 dark:text-blue-400', bg: 'bg-blue-500/15', labelKey: 'video.direction.outgoing' }
    case 'missed':
      return { icon: PhoneMissed, tone: 'text-red-600 dark:text-red-400', bg: 'bg-red-500/15', labelKey: 'video.direction.missed' }
  }
}

function triggerTextDownload(content: string, filename: string): void {
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// ---------------------------------------------------------------------------
// Info row
// ---------------------------------------------------------------------------

function InfoRow({ icon: Icon, label, value }: { icon: typeof Phone; label: string; value: string }) {
  return (
    <div className="flex items-center gap-2.5">
      <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
      <span className="w-24 shrink-0 text-xs text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-sm text-foreground">{value}</span>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function CallHistoryDetailModal({
  entry,
  open,
  onClose,
  onCallBack,
}: {
  entry: CallHistoryEntry | null
  open: boolean
  onClose: () => void
  /** Called with the contact name to prefill a new call/meeting. */
  onCallBack: (contactName: string) => void
}) {
  const { t } = useTranslation()
  if (!entry) return null

  const dir = directionMeta(entry.direction)
  const DirIcon = dir.icon
  const typeLabel = entry.type === 'video'
    ? t('video.history.detail.typeVideo')
    : t('video.history.detail.typeAudio')
  const dateLabel = formatDate(new Date(entry.date), { day: '2-digit', month: '2-digit', year: 'numeric' })

  const handleDownload = () => {
    const lines = [
      t('video.history.detail.protocolHeading'),
      '='.repeat(t('video.history.detail.protocolHeading').length),
      '',
      `${t('video.history.detail.contact')}: ${entry.contactName}`,
      entry.contactCompany ? `${t('video.history.detail.company')}: ${entry.contactCompany}` : null,
      entry.contactPhone ? `${t('video.history.detail.phone')}: ${entry.contactPhone}` : null,
      `${t('video.history.detail.type')}: ${typeLabel}`,
      `${t('video.history.detail.direction')}: ${t(dir.labelKey)}`,
      `${t('video.history.detail.date')}: ${dateLabel}`,
      `${t('video.history.detail.time')}: ${entry.startTime}`,
      `${t('video.history.detail.duration')}: ${formatDuration(entry.duration, t)}`,
      entry.topic ? `${t('video.history.detail.topic')}: ${entry.topic}` : null,
      entry.participants?.length
        ? `${t('video.history.detail.participants')}: ${entry.participants.map((p) => p.name).join(', ')}`
        : null,
      '',
      entry.notes ? `${t('video.history.detail.notes')}:` : null,
      entry.notes ?? null,
    ].filter((l): l is string => l !== null)
    triggerTextDownload(lines.join('\n'), `${t('video.history.detail.protocolFilename')}-${entry.id}.txt`)
  }

  return (
    <DetailModal
      open={open}
      onClose={onClose}
      title={entry.contactName}
      subtitle={entry.topic}
      badge={
        <span className={`ml-1 inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${dir.bg} ${dir.tone}`}>
          <DirIcon className="h-3.5 w-3.5" />
          {t(dir.labelKey)}
        </span>
      }
      footer={
        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={handleDownload}>
            <Download className="mr-1.5 h-4 w-4" />
            {t('video.history.detail.downloadProtocol')}
          </Button>
          <Button size="sm" onClick={() => { onCallBack(entry.contactName); onClose() }}>
            <PhoneCall className="mr-1.5 h-4 w-4" />
            {t('video.history.detail.callBack')}
          </Button>
        </div>
      }
    >
      <div className="space-y-5">
        {/* Kontakt-Kopf */}
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-base font-medium text-muted-foreground">
            {entry.contactInitials}
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-foreground">{entry.contactName}</p>
            <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
              {entry.type === 'video' ? <Video className="h-3.5 w-3.5" /> : <Phone className="h-3.5 w-3.5" />}
              {typeLabel}
            </p>
          </div>
        </div>

        {/* Meta-Grid */}
        <div className="grid gap-3 rounded-xl border border-border bg-card p-4 sm:grid-cols-2">
          <InfoRow icon={DirIcon} label={t('video.history.detail.direction')} value={t(dir.labelKey)} />
          <InfoRow icon={Clock} label={t('video.history.detail.duration')} value={formatDuration(entry.duration, t)} />
          <InfoRow icon={CalendarDays} label={t('video.history.detail.date')} value={`${dateLabel} · ${entry.startTime}`} />
          {entry.contactCompany && (
            <InfoRow icon={Building2} label={t('video.history.detail.company')} value={entry.contactCompany} />
          )}
          {entry.contactPhone && (
            <InfoRow icon={Phone} label={t('video.history.detail.phone')} value={entry.contactPhone} />
          )}
        </div>

        {/* Teilnehmer */}
        {entry.participants && entry.participants.length > 0 && (
          <div>
            <h4 className="mb-2 flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
              <Users className="h-3.5 w-3.5" />
              {t('video.history.detail.participants')} ({entry.participants.length})
            </h4>
            <div className="flex flex-wrap gap-2">
              {entry.participants.map((p) => (
                <span key={p.id} className="inline-flex items-center gap-1.5 rounded-full border border-border bg-secondary/40 py-1 pl-1 pr-2.5 text-xs text-foreground">
                  <span className="flex h-5 w-5 items-center justify-center rounded-full bg-muted text-[10px] font-medium text-muted-foreground">
                    {p.initials}
                  </span>
                  {p.name}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Notiz */}
        {entry.notes && (
          <div>
            <h4 className="mb-2 flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
              <FileText className="h-3.5 w-3.5" />
              {t('video.history.detail.notes')}
            </h4>
            <p className="whitespace-pre-line rounded-xl border border-border bg-secondary/30 p-3.5 text-sm leading-relaxed text-foreground">
              {entry.notes}
            </p>
          </div>
        )}

        {/* Aufzeichnung */}
        {entry.hasRecording && (
          <div className="flex items-center gap-3 rounded-xl border border-border bg-card p-4">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
              <Video className="h-5 w-5" />
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-foreground">{t('video.history.detail.recordingAvailable')}</p>
              {typeof entry.recordingDuration === 'number' && (
                <p className="text-xs text-muted-foreground">
                  {t('video.history.detail.recordingLength', { length: formatRecordingLength(entry.recordingDuration) })}
                </p>
              )}
            </div>
          </div>
        )}
      </div>
    </DetailModal>
  )
}
