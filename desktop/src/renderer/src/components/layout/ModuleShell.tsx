/**
 * Module loading utilities: loading fallback and error boundary.
 *
 * Every lazy-loaded module is wrapped in a Suspense with ModuleLoadingFallback,
 * and an error boundary (ModuleErrorBoundary) that allows retry on failure.
 */
import { Component, type ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Skeleton, SkeletonTable } from '@/components/shared'

interface ErrorFallbackProps {
  errorMessage: string | null
  onRetry: () => void
  labels: {
    title: string
    defaultError: string
    retry: string
  }
}

/** Skeleton scaffold shown while a lazy-loaded module is loading (no spinner). */
export function ModuleLoadingFallback() {
  return (
    <div className="h-full overflow-hidden p-6 space-y-6">
      {/* Header (title + description) */}
      <div className="space-y-2.5">
        <Skeleton className="h-7 w-56" />
        <Skeleton className="h-4 w-80" />
      </div>
      {/* KPI row */}
      <div className="flex flex-wrap gap-x-10 gap-y-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="space-y-2">
            <Skeleton className="h-3 w-20" />
            <Skeleton className="h-6 w-16" />
          </div>
        ))}
      </div>
      {/* Content table */}
      <SkeletonTable rows={7} cols={5} />
    </div>
  )
}

interface ErrorBoundaryProps {
  children: ReactNode
  labels: {
    title: string
    defaultError: string
    retry: string
  }
}

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
}

function ErrorFallback({ errorMessage, onRetry, labels }: ErrorFallbackProps) {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="text-center max-w-md">
        <h2 className="text-lg font-semibold text-foreground">{labels.title}</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          {errorMessage || labels.defaultError}
        </p>
        <Button className="mt-4" variant="outline" onClick={onRetry}>
          {labels.retry}
        </Button>
      </div>
    </div>
  )
}

/**
 * Error boundary that catches render errors in module content
 * and shows a retry button instead of a blank screen.
 */
class ModuleErrorBoundaryInner extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    // Log error details for debugging — structured console output
    console.error('[ModuleErrorBoundary] Uncaught render error:', {
      message: error.message,
      stack: error.stack,
      componentStack: errorInfo.componentStack,
    })
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      return (
        <ErrorFallback
          errorMessage={this.state.error?.message ?? null}
          onRetry={this.handleRetry}
          labels={this.props.labels}
        />
      )
    }

    return this.props.children
  }
}

/** Wrapper that resets the error boundary on route change via React key. */
export function ModuleErrorBoundary({ children }: { children: ReactNode }) {
  const location = useLocation()
  const { t } = useTranslation()
  const labels = {
    title: t('layout.moduleShell.errorTitle'),
    defaultError: t('layout.moduleShell.errorDefault'),
    retry: t('layout.moduleShell.retry'),
  }
  return (
    <ModuleErrorBoundaryInner key={location.pathname} labels={labels}>
      {children}
    </ModuleErrorBoundaryInner>
  )
}
