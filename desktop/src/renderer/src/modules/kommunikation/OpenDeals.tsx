import { TrendingUp, ArrowRight } from 'lucide-react'
import { formatCurrency } from '@/lib/format'

// ---------------------------------------------------------------------------
// Mock deal data (linked from conversation.crmDealIds)
// ---------------------------------------------------------------------------

interface MockDeal {
  id: string
  name: string
  value: number
  stage: string
  probability: number
}

const mockDeals: Record<string, MockDeal> = {
  d3: { id: 'd3', name: 'Website Redesign', value: 18500, stage: 'Angebot', probability: 60 },
  d5: { id: 'd5', name: 'Mobile App MVP', value: 45000, stage: 'Verhandlung', probability: 40 },
  d8: { id: 'd8', name: 'Wartungsvertrag 2026', value: 4800, stage: 'Qualifiziert', probability: 80 },
}

const stageColors: Record<string, string> = {
  Qualifiziert: 'bg-blue-500',
  Angebot: 'bg-primary',
  Verhandlung: 'bg-warning',
  Gewonnen: 'bg-success',
  Verloren: 'bg-destructive',
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface OpenDealsProps {
  dealIds: string[]
}

export function OpenDeals({ dealIds }: OpenDealsProps) {
  if (dealIds.length === 0) return null

  const deals = dealIds
    .map((id) => mockDeals[id])
    .filter(Boolean)

  if (deals.length === 0) return null

  return (
    <div className="p-3 border-b border-border">
      <div className="flex items-center gap-1.5 mb-2">
        <TrendingUp className="h-3 w-3 text-muted-foreground" />
        <h4 className="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
          Offene Deals
        </h4>
        <span className="ml-auto text-[10px] text-muted-foreground">{deals.length}</span>
      </div>
      <div className="space-y-1.5">
        {deals.map((deal) => (
          <button
            key={deal.id}
            className="flex w-full items-center gap-2 rounded-md bg-secondary/50 px-2.5 py-2 text-left hover:bg-secondary transition-colors group"
          >
            {/* Stage dot */}
            <span className={`h-2 w-2 shrink-0 rounded-full ${stageColors[deal.stage] ?? 'bg-muted-foreground'}`} />
            <div className="min-w-0 flex-1">
              <p className="text-xs font-medium text-foreground truncate">{deal.name}</p>
              <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
                <span>{deal.stage}</span>
                <span>·</span>
                <span>{formatCurrency(deal.value)}</span>
                <span>·</span>
                <span>{deal.probability}%</span>
              </div>
            </div>
            <ArrowRight className="h-3 w-3 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity shrink-0" />
          </button>
        ))}
      </div>
    </div>
  )
}
