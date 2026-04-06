/**
 * Widget wrapper -- card shell around each dashboard widget.
 *
 * Provides a drag handle, header with name and remove button (edit mode),
 * Suspense loading skeleton, and an error boundary for crash isolation.
 */
import { Suspense, Component, memo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { X, GripVertical } from 'lucide-react'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { widgetRegistry } from './WidgetRegistry'
import { useDashboardStore } from '@/stores/dashboard'

interface WidgetWrapperProps {
  widgetId: string
  isEditing: boolean
}

/** Loading skeleton shown while the lazy widget component loads. */
function WidgetSkeleton() {
  return (
    <div className="space-y-3 p-4">
      <Skeleton className="h-4 w-2/3" />
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-5/6" />
      <Skeleton className="h-4 w-1/2" />
    </div>
  )
}

/** Error boundary scoped to a single widget -- prevents one crash from breaking the whole dashboard. */
interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

class WidgetErrorBoundary extends Component<
  { children: ReactNode; onRetry: () => void; errorLabel: string; retryLabel: string },
  ErrorBoundaryState
> {
  constructor(props: { children: ReactNode; onRetry: () => void; errorLabel: string; retryLabel: string }) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
    this.props.onRetry()
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex h-full items-center justify-center p-4">
          <div className="text-center">
            <p className="text-sm text-muted-foreground">
              {this.props.errorLabel}
            </p>
            <Button
              variant="ghost"
              size="sm"
              className="mt-2"
              onClick={this.handleRetry}
            >
              {this.props.retryLabel}
            </Button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}

export const WidgetWrapper = memo(function WidgetWrapper({
  widgetId,
  isEditing,
}: WidgetWrapperProps) {
  const { t } = useTranslation()
  const definition = widgetRegistry[widgetId]
  const removeWidget = useDashboardStore((s) => s.removeWidget)

  if (!definition) return null

  const WidgetComponent = definition.component
  const Icon = definition.icon

  return (
    <Card className="flex h-full flex-col overflow-hidden">
      {/* Header with drag handle */}
      <CardHeader className="flex flex-row items-center gap-2 space-y-0 border-b px-3 py-2">
        {isEditing && (
          <div className="widget-drag-handle cursor-grab active:cursor-grabbing">
            <GripVertical className="h-4 w-4 text-muted-foreground" />
          </div>
        )}
        <Icon className="h-4 w-4 text-muted-foreground" />
        <span className="flex-1 text-sm font-medium">{definition.name}</span>
        {isEditing && (
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={(e) => {
              e.stopPropagation()
              removeWidget(widgetId)
            }}
            aria-label={t('widgets.wrapper.removeLabel', { name: definition.name })}
          >
            <X className="h-3.5 w-3.5" />
          </Button>
        )}
      </CardHeader>

      {/* Widget content */}
      <CardContent className="flex-1 overflow-auto p-0">
        <WidgetErrorBoundary
          onRetry={() => { /* re-render triggers refetch */ }}
          errorLabel={t('widgets.wrapper.errorLoading')}
          retryLabel={t('common.retry')}
        >
          <Suspense fallback={<WidgetSkeleton />}>
            <WidgetComponent
              widgetId={widgetId}
              isEditing={isEditing}
            />
          </Suspense>
        </WidgetErrorBoundary>
      </CardContent>
    </Card>
  )
})
