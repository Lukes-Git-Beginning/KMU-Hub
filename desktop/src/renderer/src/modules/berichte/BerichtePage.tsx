import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  BarChart3,
  CalendarClock,
  FileText,
  Landmark,
  Plus,
} from 'lucide-react'
import { PageHeader } from '@/components/shared'
import {
  useCreateReportDocument,
  useDashboardKPIs,
  useDefinitions,
  useSchedules,
} from '@/api/hooks/useBerichte'
import { DashboardGrid } from './components/DashboardGrid'
import { PinnedReports } from './components/PinnedReports'
import { BerichtLibrary } from './components/documents/BerichtLibrary'
import { ReportDocumentEditor } from './components/documents/ReportDocumentEditor'
import { ScheduleList } from './components/ScheduleList'
import { DatevView } from './components/DatevView'

type TabKey = 'dashboard' | 'berichte' | 'geplant' | 'datev'

const MODULE_OPTIONS: { id: string; name: string }[] = [
  { id: 'finanzen', name: 'Finanzen' },
  { id: 'crm', name: 'CRM / Kontakte' },
  { id: 'helpdesk', name: 'Helpdesk' },
  { id: 'inventar', name: 'Inventar' },
  { id: 'produktion', name: 'Produktion' },
  { id: 'cross', name: 'Bereichsübergreifend' },
]

export default function BerichtePage() {
  const { t } = useTranslation()

  const [tab, setTab] = useState<TabKey>('dashboard')
  const [moduleFilter, setModuleFilter] = useState<string>('all')
  const [openDocId, setOpenDocId] = useState<string | null>(null)

  const kpisQuery = useDashboardKPIs(
    moduleFilter === 'all' ? undefined : [moduleFilter],
  )
  const definitionsQuery = useDefinitions({ kind: 'system', is_published: true })
  const schedulesQuery = useSchedules()
  const createDoc = useCreateReportDocument()

  const kpis = kpisQuery.data?.kpis ?? []
  const definitions = definitionsQuery.data?.definitions ?? []
  const schedules = useMemo(
    () => schedulesQuery.data?.schedules ?? [],
    [schedulesQuery.data],
  )

  const activeScheduled = useMemo(
    () => schedules.filter((s) => s.active).length,
    [schedules],
  )

  const openCreated = (id: string) => {
    setTab('berichte')
    setOpenDocId(id)
  }

  const handleNewReport = () => {
    createDoc.mutate(
      { title: t('berichte.docs.newTitle'), module: 'cross' },
      { onSuccess: (res) => openCreated(res.document.id) },
    )
  }

  const handleNewFromTemplate = (templateId: string) => {
    createDoc.mutate(
      { template_id: templateId },
      { onSuccess: (res) => openCreated(res.document.id) },
    )
  }

  // Editor takes over the full module area (its own header + back button).
  if (openDocId) {
    return (
      <ReportDocumentEditor documentId={openDocId} onBack={() => setOpenDocId(null)} />
    )
  }

  const tabs = [
    { key: 'dashboard' as const, label: t('berichte.tabs.dashboard'), icon: BarChart3 },
    { key: 'berichte' as const, label: t('berichte.tabs.berichte'), icon: FileText },
    {
      key: 'geplant' as const,
      label: t('berichte.tabs.geplant', { count: schedules.length }),
      icon: CalendarClock,
    },
    { key: 'datev' as const, label: 'DATEV', icon: Landmark },
  ]

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <PageHeader
        title={t('berichte.page.title')}
        description={t('berichte.page.description', {
          kpis: kpis.length,
          scheduled: activeScheduled,
        })}
        icon={BarChart3}
        moduleId="berichte"
        actions={
          <button
            onClick={handleNewReport}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-sm text-primary-foreground transition-colors hover:bg-button-primary-hover"
          >
            <Plus className="h-4 w-4" />
            {t('berichte.actions.neuerBericht')}
          </button>
        }
        className="mb-6"
      />

      <div className="mb-6 flex items-center gap-4 border-b border-border">
        {tabs.map((tabItem) => (
          <button
            key={tabItem.key}
            onClick={() => setTab(tabItem.key)}
            className={`flex items-center gap-1.5 border-b-2 px-1 pb-2 text-sm transition-colors ${
              tab === tabItem.key
                ? 'border-primary font-medium text-primary tab-accent-active'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <tabItem.icon className="h-3.5 w-3.5" />
            {tabItem.label}
          </button>
        ))}
      </div>

      {tab === 'dashboard' && (
        <>
          <PinnedReports />
          <DashboardGrid
            kpis={kpis}
            definitions={definitions}
            moduleFilter={moduleFilter}
            onModuleFilterChange={setModuleFilter}
            moduleOptions={MODULE_OPTIONS}
            isLoading={kpisQuery.isLoading || definitionsQuery.isLoading}
          />
        </>
      )}

      {tab === 'berichte' && (
        <BerichtLibrary
          onOpen={(doc) => setOpenDocId(doc.id)}
          onNew={handleNewReport}
          onNewFromTemplate={handleNewFromTemplate}
        />
      )}

      {tab === 'geplant' && <ScheduleList definitions={definitions} />}

      {tab === 'datev' && <DatevView definitions={definitions} />}
    </div>
  )
}
