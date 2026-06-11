"use memo"
import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Pencil, Check, RotateCcw, User, Users } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { AlertsSection, ModulesGrid } from '@/components/dashboard'
import WidgetContainer from '@/components/widgets/WidgetContainer'
import { useDashboardStore, type DashboardScope } from '@/stores/dashboard'
import { useDashboardPrefsStore } from '@/stores/dashboardPrefs'
import { QuickActionsBar } from '@/components/dashboard/QuickActionsBar'
import { ProfileWidgetSuggestions } from '@/components/dashboard/ProfileWidgetSuggestions'
import { TextReveal } from '@/components/shared/TextReveal'

function getGreetingKey() {
  const hour = new Date().getHours()
  if (hour < 12) return 'dashboard.greeting.morning' as const
  if (hour < 18) return 'dashboard.greeting.afternoon' as const
  return 'dashboard.greeting.evening' as const
}

// ── Segmented Control ──────────────────────────────────────────────────────

interface ScopeToggleProps {
  scope: DashboardScope
  onChange: (scope: DashboardScope) => void
}

function ScopeToggle({ scope, onChange }: ScopeToggleProps) {
  const { t } = useTranslation()

  return (
    <div
      className="inline-flex items-center rounded-lg border border-border bg-muted/50 p-0.5"
      role="group"
      aria-label={t('dashboard.scope.label')}
    >
      <button
        type="button"
        role="radio"
        aria-checked={scope === 'personal'}
        onClick={() => onChange('personal')}
        className={`
          flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors
          ${scope === 'personal'
            ? 'bg-background text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground'
          }
        `}
        data-testid="scope-toggle-personal"
      >
        <User className="h-3.5 w-3.5" />
        {t('dashboard.scope.personal')}
      </button>
      <button
        type="button"
        role="radio"
        aria-checked={scope === 'team'}
        onClick={() => onChange('team')}
        className={`
          flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors
          ${scope === 'team'
            ? 'bg-background text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground'
          }
        `}
        data-testid="scope-toggle-team"
      >
        <Users className="h-3.5 w-3.5" />
        {t('dashboard.scope.team')}
      </button>
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const { t } = useTranslation()
  const greetingKey = useMemo(() => getGreetingKey(), [])

  const scope = useDashboardStore((s) => s.scope)
  const setScope = useDashboardStore((s) => s.setScope)
  const isEditing = useDashboardStore((s) => s.isEditing)
  const toggleEditing = useDashboardStore((s) => s.toggleEditing)
  const resetToDefaults = useDashboardStore((s) => s.resetToDefaults)
  const ensureDefaults = useDashboardStore((s) => s.ensureDefaults)
  const showGreeting = useDashboardPrefsStore((s) => s.showGreeting)

  useEffect(() => {
    ensureDefaults()
  }, [ensureDefaults])

  return (
    <div className="h-full overflow-auto">
      <div className="p-4 md:p-8">
        {/* Greeting Header — text part is a personal pref (settings panel) */}
        <div className="mb-8 flex items-start justify-between animate-fade-up">
          <div>
            {showGreeting && scope === 'personal' && (
              <>
                <h1 className="text-2xl font-semibold text-foreground">
                  <TextReveal text={t(greetingKey)} wordDelay={80} />
                </h1>
                <p className="mt-1 text-sm text-muted-foreground animate-fade-up" style={{ animationDelay: '200ms' }}>
                  {t('dashboard.greeting.subtitle')}
                </p>
              </>
            )}
            {scope === 'team' && (
              <div>
                <h1 className="text-2xl font-semibold text-foreground">
                  {t('dashboard.scope.teamHeadline')}
                </h1>
                <p className="mt-1 text-sm text-muted-foreground">
                  {t('dashboard.scope.teamSubline')}
                </p>
              </div>
            )}
          </div>

          {/* Dashboard edit controls + scope toggle */}
          <div className="flex items-center gap-3">
            <ScopeToggle scope={scope} onChange={setScope} />

            <div className="flex items-center gap-2">
              {isEditing && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={resetToDefaults}
                  className="text-muted-foreground"
                >
                  <RotateCcw className="mr-1.5 h-3.5 w-3.5" />
                  {t('dashboard.edit.reset')}
                </Button>
              )}
              <Button
                variant={isEditing ? 'default' : 'outline'}
                size="sm"
                onClick={toggleEditing}
              >
                {isEditing ? (
                  <>
                    <Check className="mr-1.5 h-3.5 w-3.5" />
                    {t('dashboard.edit.done')}
                  </>
                ) : (
                  <>
                    <Pencil className="mr-1.5 h-3.5 w-3.5" />
                    {t('dashboard.edit.customize')}
                  </>
                )}
              </Button>
            </div>
          </div>
        </div>

        {/* Personal-scope-only sections */}
        {scope === 'personal' && (
          <>
            {/* Alerts */}
            <div className="animate-fade-up stagger-1">
              <AlertsSection />
            </div>

            {/* Profile Widget Suggestions */}
            <div className="animate-fade-up stagger-2">
              <ProfileWidgetSuggestions />
            </div>

            {/* Modules Grid */}
            <div className="animate-fade-up stagger-3">
              <ModulesGrid />
            </div>
          </>
        )}

        {/* Widget Grid — shared, scope-aware via store */}
        <div className="mb-8 animate-fade-up stagger-4">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-foreground">
              {scope === 'team'
                ? t('dashboard.scope.teamWidgetsTitle')
                : t('dashboard.widgets.title')}
            </h2>
            {isEditing && (
              <p className="text-xs text-muted-foreground">
                {t('dashboard.widgets.editHint')}
              </p>
            )}
          </div>
          {scope === 'personal' && (
            <div className="mb-6">
              <QuickActionsBar />
            </div>
          )}
          <WidgetContainer />
        </div>
      </div>
    </div>
  )
}
