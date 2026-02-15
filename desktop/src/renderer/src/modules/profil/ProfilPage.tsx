import { lazy, Suspense, useState } from 'react'
import { User, Clock, Calendar, FolderOpen } from 'lucide-react'
import { cn } from '@/lib/cn'

const ProfilTab = lazy(() => import('./tabs/ProfilTab'))
const ZeiterfassungTab = lazy(() => import('./tabs/ZeiterfassungTab'))
const AbwesenheitenTab = lazy(() => import('./tabs/AbwesenheitenTab'))
const DokumenteTab = lazy(() => import('./tabs/DokumenteTab'))

type TabKey = 'profil' | 'zeiterfassung' | 'abwesenheiten' | 'dokumente'

const TABS: { key: TabKey; label: string; icon: typeof User }[] = [
  { key: 'profil', label: 'Mein Profil', icon: User },
  { key: 'zeiterfassung', label: 'Zeiterfassung', icon: Clock },
  { key: 'abwesenheiten', label: 'Abwesenheiten', icon: Calendar },
  { key: 'dokumente', label: 'Dokumente', icon: FolderOpen },
]

export default function ProfilPage() {
  const [activeTab, setActiveTab] = useState<TabKey>('profil')

  return (
    <div className="h-full flex flex-col overflow-hidden">
      {/* Tab Header */}
      <div className="border-b border-border bg-card/50 px-6">
        <nav className="flex gap-1" role="tablist">
          {TABS.map((tab) => {
            const Icon = tab.icon
            const isActive = activeTab === tab.key
            return (
              <button
                key={tab.key}
                role="tab"
                aria-selected={isActive}
                onClick={() => setActiveTab(tab.key)}
                className={cn(
                  'flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
                  isActive
                    ? 'border-primary text-primary'
                    : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border',
                )}
              >
                <Icon className="h-4 w-4" />
                {tab.label}
              </button>
            )
          })}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-auto">
        <Suspense
          fallback={
            <div className="flex items-center justify-center h-64">
              <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
            </div>
          }
        >
          {activeTab === 'profil' && <ProfilTab />}
          {activeTab === 'zeiterfassung' && <ZeiterfassungTab />}
          {activeTab === 'abwesenheiten' && <AbwesenheitenTab />}
          {activeTab === 'dokumente' && <DokumenteTab />}
        </Suspense>
      </div>
    </div>
  )
}
