/**
 * Beratungsprotokolle history — rendered inside the contact detail modal.
 *
 * Lists the contact's advisory protocols (newest first) and opens the full-screen
 * editor on click / "new". Creating a draft happens here so the editor route
 * always loads an existing record.
 */
import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Plus, Lock, FileText, ClipboardList } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useAdvisoryProtocolsStore } from '@/stores/advisoryProtocols'
import {
  ADVISORY_OCCASIONS,
  SRI_CLASSES,
  sriColor,
} from '@/lib/advisory'
import { formatDate } from '@/lib/format'

interface AdvisoryProtocolsTabProps {
  contactId: string
  advisorName?: string
  onNavigate: () => void
}

export function AdvisoryProtocolsTab({ contactId, advisorName, onNavigate }: AdvisoryProtocolsTabProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  // Select the stable array, then derive — selecting s.byContact(id) would return
  // a fresh array each render and trigger an infinite re-render loop.
  const allProtocols = useAdvisoryProtocolsStore((s) => s.protocols)
  const createDraft = useAdvisoryProtocolsStore((s) => s.createDraft)
  const protocols = useMemo(
    () =>
      allProtocols
        .filter((p) => p.contactId === contactId)
        .sort((a, b) => (b.date || b.createdAt).localeCompare(a.date || a.createdAt)),
    [allProtocols, contactId],
  )

  const open = (id: string) => {
    onNavigate()
    navigate(`/kontakte/protokoll/${contactId}/${id}`)
  }

  const handleNew = () => {
    const id = createDraft(contactId, advisorName ?? '')
    open(id)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs text-muted-foreground">{t('advisory.tab.subtitle')}</p>
        <Button size="sm" onClick={handleNew}>
          <Plus className="mr-2 h-4 w-4" />
          {t('advisory.tab.new')}
        </Button>
      </div>

      {protocols.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-2xl border border-dashed border-border py-12 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
            <ClipboardList className="h-6 w-6 text-primary" />
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium text-foreground">{t('advisory.tab.emptyTitle')}</p>
            <p className="text-xs text-muted-foreground">{t('advisory.tab.emptyBody')}</p>
          </div>
        </div>
      ) : (
        <ul className="space-y-2">
          {protocols.map((p) => {
            const occasion = ADVISORY_OCCASIONS.find((o) => o.value === p.occasion)
            const sri = SRI_CLASSES.find((c) => c.value === p.riskClass)
            return (
              <li key={p.id}>
                <button
                  onClick={() => open(p.id)}
                  className="flex w-full items-center gap-3 rounded-xl border border-border bg-card px-4 py-3 text-left transition-colors hover:bg-accent"
                >
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-secondary">
                    <FileText className="h-4.5 w-4.5 text-muted-foreground" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium text-foreground">
                        {p.date ? formatDate(p.date) : t('advisory.tab.undated')}
                      </span>
                      {occasion && (
                        <span className="text-xs text-muted-foreground">· {t(occasion.labelKey)}</span>
                      )}
                    </div>
                    <p className="truncate text-xs text-muted-foreground">
                      {p.advisor || t('advisory.tab.noAdvisor')}
                    </p>
                  </div>
                  <span className="flex shrink-0 items-center gap-2">
                    {sri && (
                      <Badge variant="outline" className="gap-1 text-[10px]">
                        <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: sriColor(p.riskClass) }} />
                        SRI {p.riskClass}
                      </Badge>
                    )}
                    {p.status === 'finalized' ? (
                      <Badge variant="secondary" className="gap-1 text-[10px]">
                        <Lock className="h-2.5 w-2.5" />
                        {t('advisory.status.finalized')}
                      </Badge>
                    ) : (
                      <Badge variant="outline" className="text-[10px]">{t('advisory.status.draft')}</Badge>
                    )}
                  </span>
                </button>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
