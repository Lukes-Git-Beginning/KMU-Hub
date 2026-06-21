import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Phone, Building2, Hash, Repeat, ExternalLink, PhoneCall } from 'lucide-react'
import { DetailModal } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { formatRelativeTime } from '@/lib/format'
import { useContactCalls, useOutcomes } from '@/api/hooks/useDialer'
import type { CampaignContact } from '@/api/dialer-client'
import { contactStatusConfig } from './ContactQueueTable'

function formatDuration(totalSeconds: number): string {
  const mm = Math.floor(totalSeconds / 60)
  const ss = String(totalSeconds % 60).padStart(2, '0')
  return `${mm}:${ss}`
}

interface ContactDetailModalProps {
  contact: CampaignContact | null
  open: boolean
  onClose: () => void
}

export default function ContactDetailModal({ contact, open, onClose }: ContactDetailModalProps) {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { data: callsData, isLoading } = useContactCalls(open ? contact?.id : undefined)
  const { data: outcomesData } = useOutcomes(true)

  if (!contact) return null

  const calls = callsData?.calls ?? []
  const statusCfg = contactStatusConfig[contact.status] ?? contactStatusConfig[1]
  const lastOutcome = outcomesData?.outcomes.find((o) => o.id === contact.outcome_id) ?? null

  const statusBadge = (
    <span
      className="inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium"
      style={{
        backgroundColor: `color-mix(in srgb, ${statusCfg.color} 12%, transparent)`,
        color: statusCfg.color,
        borderColor: `color-mix(in srgb, ${statusCfg.color} 30%, transparent)`,
      }}
    >
      {t(statusCfg.key)}
    </span>
  )

  return (
    <DetailModal
      open={open}
      onClose={onClose}
      title={contact.contact_name}
      subtitle={contact.contact_company ?? undefined}
      badge={statusBadge}
      footer={
        contact.contact_id ? (
          <div className="flex justify-end">
            <Button
              variant="outline"
              size="sm"
              className="gap-2"
              onClick={() => navigate(`/kontakte?contact=${contact.contact_id}`)}
            >
              <ExternalLink className="h-4 w-4" />
              {t('dialer.workspace.openInCrm')}
            </Button>
          </div>
        ) : undefined
      }
    >
      <div className="space-y-6">
        {/* Contact facts */}
        <div className="grid grid-cols-2 gap-3">
          <Fact icon={Phone} value={contact.contact_phone} mono />
          <Fact icon={Building2} value={contact.contact_company ?? '—'} />
          <Fact icon={Hash} value={`#${contact.position}`} />
          <Fact
            icon={Repeat}
            value={t('dialer.contactDetail.attempts', { count: contact.call_count })}
          />
        </div>

        {/* Last outcome */}
        {lastOutcome && (
          <section className="space-y-2">
            <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t('dialer.contactDetail.lastOutcome')}
            </h4>
            <div className="flex items-center gap-2 rounded-lg border border-border bg-secondary/40 px-3 py-2">
              <span className="h-3 w-3 rounded-full" style={{ backgroundColor: lastOutcome.color }} />
              <span className="text-sm font-medium text-foreground">{lastOutcome.label}</span>
            </div>
            {contact.notes && (
              <p className="rounded-lg bg-secondary/40 px-3 py-2 text-sm text-muted-foreground">
                {contact.notes}
              </p>
            )}
          </section>
        )}

        {/* Call history */}
        <section className="space-y-2">
          <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            {t('dialer.contactDetail.callHistory')}
          </h4>
          {isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-14 w-full rounded-lg" />
              ))}
            </div>
          ) : calls.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-8 text-center">
              <PhoneCall className="mb-2 h-7 w-7 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">{t('dialer.contactDetail.noCalls')}</p>
            </div>
          ) : (
            <ul className="space-y-2">
              {calls.map((call) => (
                <li
                  key={call.id}
                  className="flex items-start gap-3 rounded-lg border border-border bg-card px-3 py-2.5"
                >
                  <span
                    className="mt-1 h-2.5 w-2.5 shrink-0 rounded-full"
                    style={{ backgroundColor: call.outcome_color }}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-sm font-medium text-foreground">
                        {call.outcome_label ?? t('dialer.workspace.wrapUp.completeWithoutOutcome')}
                      </span>
                      <span className="font-mono text-xs tabular-nums text-muted-foreground">
                        {formatDuration(call.duration_seconds)}
                      </span>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {call.agent_name} · {formatRelativeTime(call.created_at, i18n.language)}
                    </p>
                    {call.notes && (
                      <p className="mt-1 text-sm text-muted-foreground">{call.notes}</p>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </DetailModal>
  )
}

function Fact({
  icon: Icon,
  value,
  mono,
}: {
  icon: typeof Phone
  value: string
  mono?: boolean
}) {
  return (
    <div className="flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-2">
      <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
      <span className={`truncate text-sm text-foreground ${mono ? 'font-mono' : ''}`}>{value}</span>
    </div>
  )
}
