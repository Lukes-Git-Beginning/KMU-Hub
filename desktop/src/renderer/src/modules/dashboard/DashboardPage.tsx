"use memo"
import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Pencil, Check, RotateCcw } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { AlertsSection, ModulesGrid } from '@/components/dashboard'
import WidgetContainer from '@/components/widgets/WidgetContainer'
import { useDashboardStore } from '@/stores/dashboard'
import { QuickActionsBar } from '@/components/dashboard/QuickActionsBar'
import { ProfileWidgetSuggestions } from '@/components/dashboard/ProfileWidgetSuggestions'
import { TextReveal } from '@/components/shared/TextReveal'

function getGreetingKey(): string {
  const hour = new Date().getHours()
  if (hour < 12) return 'dashboard.greeting.morning'
  if (hour < 18) return 'dashboard.greeting.afternoon'
  return 'dashboard.greeting.evening'
}

export default function DashboardPage() {
  const { t } = useTranslation()
  const greetingKey = useMemo(() => getGreetingKey(), [])
  const isEditing = useDashboardStore((s) => s.isEditing)
  const toggleEditing = useDashboardStore((s) => s.toggleEditing)
  const resetToDefaults = useDashboardStore((s) => s.resetToDefaults)
  const ensureDefaults = useDashboardStore((s) => s.ensureDefaults)

  useEffect(() => {
    ensureDefaults()
  }, [ensureDefaults])

  return (
    <div className="h-full overflow-auto">
      <div className="p-4 md:p-8">
        {/* Greeting Header */}
        <div className="mb-8 flex items-start justify-between animate-fade-up">
          <div>
            <h1 className="text-2xl font-semibold text-foreground">
              <TextReveal text={t(greetingKey)} wordDelay={80} />
            </h1>
            <p className="mt-1 text-sm text-muted-foreground animate-fade-up" style={{ animationDelay: '200ms' }}>
              {t('dashboard.greeting.subtitle')}
            </p>
          </div>

          {/* Dashboard edit controls */}
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

        {/* Widget Grid */}
        <div className="mb-8 animate-fade-up stagger-4">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-foreground">
              {t('dashboard.widgets.title')}
            </h2>
            {isEditing && (
              <p className="text-xs text-muted-foreground">
                {t('dashboard.widgets.editHint')}
              </p>
            )}
          </div>
          <div className="mb-6">
            <QuickActionsBar />
          </div>
          <WidgetContainer />
        </div>
      </div>
    </div>
  )
}
