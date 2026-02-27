import { cn } from '@/lib'

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
  label?: string
}

const sizes = {
  sm: 'h-8 w-8',
  md: 'h-12 w-12',
  lg: 'h-16 w-16',
}

const textSizes = {
  sm: 'text-[6px]',
  md: 'text-[8px]',
  lg: 'text-[10px]',
}

export function LoadingSpinner({
  size = 'md',
  className,
  label,
}: LoadingSpinnerProps) {
  return (
    <div
      className={cn('flex flex-col items-center justify-center gap-3', className)}
      role="status"
      aria-live="polite"
      aria-label={label || 'Wird geladen'}
    >
      <div className={cn('relative', sizes[size])} aria-hidden="true">
        {/* Outer ring — gradient stroke, spinning */}
        <svg
          className="absolute inset-0 h-full w-full animate-spin-smooth"
          viewBox="0 0 48 48"
          fill="none"
        >
          <circle
            cx="24"
            cy="24"
            r="20"
            stroke="var(--border)"
            strokeWidth="3"
          />
          <path
            d="M24 4 a20 20 0 0 1 20 20"
            stroke="url(#spinner-gradient)"
            strokeWidth="3"
            strokeLinecap="round"
          />
          <defs>
            <linearGradient id="spinner-gradient" x1="24" y1="4" x2="44" y2="24" gradientUnits="userSpaceOnUse">
              <stop stopColor="var(--accent-1)" />
              <stop offset="1" stopColor="var(--accent-2)" />
            </linearGradient>
          </defs>
        </svg>

        {/* Center — KMU branding text */}
        <div className="absolute inset-0 flex items-center justify-center">
          <span
            className={cn(
              'font-bold tracking-wider text-primary select-none',
              textSizes[size]
            )}
          >
            KMU
          </span>
        </div>
      </div>

      {label && (
        <span className="text-sm text-muted-foreground animate-fade-in">{label}</span>
      )}
    </div>
  )
}
