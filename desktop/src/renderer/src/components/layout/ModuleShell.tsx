/**
 * Module loading utilities: loading fallback and error boundary.
 *
 * Every lazy-loaded module is wrapped in a Suspense with ModuleLoadingFallback,
 * and an error boundary (ModuleErrorBoundary) that allows retry on failure.
 */
import { Component, type ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { Button } from '@/components/ui/button'

/** Centered spinner shown while a lazy-loaded module is loading. */
export function ModuleLoadingFallback() {
  return (
    <div className="flex h-full items-center justify-center">
      <div className="text-center">
        <div className="mx-auto h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
        <p className="mt-4 text-sm text-muted-foreground">Laden...</p>
      </div>
    </div>
  )
}

interface ErrorBoundaryProps {
  children: ReactNode
}

interface ErrorBoundaryState {
  hasError: boolean
  error: Error | null
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
        <div className="flex h-full items-center justify-center">
          <div className="text-center max-w-md">
            <h2 className="text-lg font-semibold text-foreground">
              Etwas ist schiefgelaufen
            </h2>
            <p className="mt-2 text-sm text-muted-foreground">
              {this.state.error?.message || 'Ein unerwarteter Fehler ist aufgetreten.'}
            </p>
            <Button
              className="mt-4"
              variant="outline"
              onClick={this.handleRetry}
            >
              Erneut versuchen
            </Button>
          </div>
        </div>
      )
    }

    return this.props.children
  }
}

/** Wrapper that resets the error boundary on route change via React key. */
export function ModuleErrorBoundary({ children }: ErrorBoundaryProps) {
  const location = useLocation()
  return (
    <ModuleErrorBoundaryInner key={location.pathname}>
      {children}
    </ModuleErrorBoundaryInner>
  )
}
