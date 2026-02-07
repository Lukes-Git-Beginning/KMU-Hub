/**
 * Deals list page with toggle between table view and pipeline (Kanban) view.
 *
 * Table view shows deals with columns for name, stage, value, contact,
 * expected close date, and tags. Pipeline view renders the DealPipelineView
 * component with horizontal stage columns.
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Plus,
  Search,
  ChevronLeft,
  ChevronRight,
  LayoutList,
  Columns3,
  TrendingUp,
} from 'lucide-react'
import { useDeals } from '@/api/hooks/useDeals'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import DealPipelineView from './DealPipelineView'

const PAGE_SIZE = 20

function formatCurrency(value?: number, currency?: string): string {
  if (value == null) return '-'
  return new Intl.NumberFormat('de-DE', {
    style: 'currency',
    currency: currency || 'EUR',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

export default function DealsListPage() {
  const navigate = useNavigate()
  const [view, setView] = useState<'list' | 'pipeline'>('list')
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search)
      setPage(1)
    }, 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading, error, refetch } = useDeals({
    page,
    page_size: PAGE_SIZE,
    search: debouncedSearch || undefined,
  })

  const deals = data?.deals ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  function showComingSoon() {
    alert('Kommt bald')
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            Fehler beim Laden der Deals
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : 'Ein unerwarteter Fehler ist aufgetreten.'}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            Erneut versuchen
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Deals</h1>
        <div className="flex items-center gap-2">
          {/* View toggle */}
          <div className="flex items-center rounded-md border border-border">
            <Button
              variant={view === 'list' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-r-none"
              onClick={() => setView('list')}
            >
              <LayoutList className="h-4 w-4" />
            </Button>
            <Button
              variant={view === 'pipeline' ? 'secondary' : 'ghost'}
              size="sm"
              className="rounded-l-none"
              onClick={() => setView('pipeline')}
            >
              <Columns3 className="h-4 w-4" />
            </Button>
          </div>
          <Button onClick={showComingSoon} className="gap-2">
            <Plus className="h-4 w-4" />
            Neuer Deal
          </Button>
        </div>
      </div>

      {view === 'pipeline' ? (
        <DealPipelineView />
      ) : (
        <>
          {/* Search bar */}
          <div className="relative max-w-sm">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Deals suchen..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>

          {isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 8 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : deals.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16">
              <TrendingUp className="h-12 w-12 text-muted-foreground" />
              <p className="mt-4 text-lg font-medium text-foreground">
                Keine Deals gefunden
              </p>
              <p className="mt-1 text-sm text-muted-foreground">
                {debouncedSearch
                  ? 'Versuche einen anderen Suchbegriff.'
                  : 'Erstelle deinen ersten Deal, um loszulegen.'}
              </p>
              {!debouncedSearch && (
                <Button className="mt-4 gap-2" onClick={showComingSoon}>
                  <Plus className="h-4 w-4" />
                  Ersten Deal erstellen
                </Button>
              )}
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Phase</TableHead>
                    <TableHead>Wert</TableHead>
                    <TableHead>Kontakt</TableHead>
                    <TableHead>Erwarteter Abschluss</TableHead>
                    <TableHead>Tags</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {deals.map((deal) => (
                    <TableRow
                      key={deal.id}
                      className="cursor-pointer"
                      onClick={() => navigate(`/crm/deals/${deal.id}`)}
                    >
                      <TableCell className="font-medium">
                        {deal.name}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{deal.stageName || '-'}</Badge>
                      </TableCell>
                      <TableCell className="font-medium">
                        {formatCurrency(deal.value, deal.currency)}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {deal.contactName || '-'}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {deal.expectedCloseDate
                          ? new Date(deal.expectedCloseDate).toLocaleDateString(
                              'de-DE'
                            )
                          : '-'}
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {deal.tags?.map((tag) => (
                            <Badge
                              key={tag.id}
                              variant="secondary"
                              className="text-xs"
                              style={
                                tag.color
                                  ? {
                                      backgroundColor: `${tag.color}20`,
                                      color: tag.color,
                                      borderColor: tag.color,
                                    }
                                  : undefined
                              }
                            >
                              {tag.name}
                            </Badge>
                          ))}
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              <div className="flex items-center justify-between">
                <p className="text-sm text-muted-foreground">
                  {total} Deal{total !== 1 ? 's' : ''} gesamt
                </p>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page <= 1}
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                  >
                    <ChevronLeft className="h-4 w-4" />
                  </Button>
                  <span className="text-sm text-muted-foreground">
                    Seite {page} von {totalPages}
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page >= totalPages}
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                  >
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </>
          )}
        </>
      )}
    </div>
  )
}
