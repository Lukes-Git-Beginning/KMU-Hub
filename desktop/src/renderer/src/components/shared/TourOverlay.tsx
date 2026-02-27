/**
 * Tour Overlay — shows a spotlight + popover for guided tours.
 *
 * When a tour is active, renders a dark overlay with a cutout around the
 * highlighted element, plus a floating popover with step info and controls.
 * If no target element is found, shows a centered modal.
 */
import { useEffect, useState, useCallback } from 'react'
import { X, ChevronLeft, ChevronRight } from 'lucide-react'
import { useTourStore } from '@/stores/tour'

export function TourOverlay() {
  const activeTour = useTourStore((s) => s.activeTour)
  const currentStep = useTourStore((s) => s.currentStep)
  const nextStep = useTourStore((s) => s.nextStep)
  const prevStep = useTourStore((s) => s.prevStep)
  const skipTour = useTourStore((s) => s.skipTour)
  const goToStep = useTourStore((s) => s.goToStep)

  const [targetRect, setTargetRect] = useState<DOMRect | null>(null)

  const step = activeTour?.steps[currentStep] ?? null
  const totalSteps = activeTour?.steps.length ?? 0
  const isLast = currentStep === totalSteps - 1

  // Find and measure target element
  const measureTarget = useCallback(() => {
    if (!step?.target) {
      setTargetRect(null)
      return
    }
    const el = document.querySelector(step.target)
    if (el) {
      setTargetRect(el.getBoundingClientRect())
    } else {
      setTargetRect(null)
    }
  }, [step?.target])

  useEffect(() => {
    measureTarget()
    window.addEventListener('resize', measureTarget)
    window.addEventListener('scroll', measureTarget, true)
    return () => {
      window.removeEventListener('resize', measureTarget)
      window.removeEventListener('scroll', measureTarget, true)
    }
  }, [measureTarget])

  // Keyboard navigation
  useEffect(() => {
    if (!activeTour) return
    const handle = (e: KeyboardEvent) => {
      if (e.key === 'Escape') skipTour()
      if (e.key === 'ArrowRight' || e.key === 'Enter') nextStep()
      if (e.key === 'ArrowLeft') prevStep()
    }
    document.addEventListener('keydown', handle)
    return () => document.removeEventListener('keydown', handle)
  }, [activeTour, nextStep, prevStep, skipTour])

  if (!activeTour || !step) return null

  // Calculate popover position
  const padding = 12
  const hasTarget = !!targetRect

  return (
    <div className="fixed inset-0 z-[9999]" onClick={skipTour}>
      {/* Dark overlay with cutout */}
      {hasTarget && targetRect ? (
        <svg className="absolute inset-0 w-full h-full">
          <defs>
            <mask id="tour-mask">
              <rect width="100%" height="100%" fill="white" />
              <rect
                x={targetRect.left - padding}
                y={targetRect.top - padding}
                width={targetRect.width + padding * 2}
                height={targetRect.height + padding * 2}
                rx="8"
                fill="black"
              />
            </mask>
          </defs>
          <rect width="100%" height="100%" fill="rgba(0,0,0,0.5)" mask="url(#tour-mask)" />
        </svg>
      ) : (
        <div className="absolute inset-0 bg-black/50" />
      )}

      {/* Popover */}
      <div
        onClick={(e) => e.stopPropagation()}
        className={`absolute bg-card border border-border rounded-xl shadow-2xl p-5 w-80 animate-fade-up ${
          hasTarget ? '' : 'left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2'
        }`}
        style={
          hasTarget && targetRect
            ? getPopoverPosition(targetRect, step.placement ?? 'bottom')
            : undefined
        }
      >
        {/* Close button */}
        <button
          onClick={skipTour}
          className="absolute top-3 right-3 rounded-md p-1 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
        >
          <X className="h-4 w-4" />
        </button>

        {/* Tour name badge */}
        <div className="text-[10px] uppercase tracking-wider text-primary font-semibold mb-2">
          {activeTour.name}
        </div>

        {/* Step content */}
        <h3 className="text-sm font-semibold text-foreground mb-1.5">{step.title}</h3>
        <p className="text-xs text-muted-foreground leading-relaxed mb-4">{step.description}</p>

        {/* Step dots */}
        <div className="flex items-center gap-1.5 mb-3">
          {Array.from({ length: totalSteps }, (_, i) => (
            <button
              key={i}
              onClick={() => goToStep(i)}
              className={`h-1.5 rounded-full transition-all ${
                i === currentStep ? 'w-4 bg-primary' : 'w-1.5 bg-border hover:bg-muted-foreground'
              }`}
            />
          ))}
          <span className="ml-auto text-[10px] text-muted-foreground tabular-nums">
            {currentStep + 1}/{totalSteps}
          </span>
        </div>

        {/* Navigation buttons */}
        <div className="flex items-center gap-2">
          <button
            onClick={skipTour}
            className="text-xs text-muted-foreground hover:text-foreground transition-colors"
          >
            Ueberspringen
          </button>
          <div className="flex-1" />
          {currentStep > 0 && (
            <button
              onClick={prevStep}
              className="flex items-center gap-1 rounded-lg border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <ChevronLeft className="h-3 w-3" />
              Zurueck
            </button>
          )}
          <button
            onClick={nextStep}
            className="flex items-center gap-1 rounded-lg bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors"
          >
            {isLast ? 'Fertig' : 'Weiter'}
            {!isLast && <ChevronRight className="h-3 w-3" />}
          </button>
        </div>
      </div>
    </div>
  )
}

function getPopoverPosition(
  rect: DOMRect,
  placement: 'top' | 'bottom' | 'left' | 'right',
): React.CSSProperties {
  const gap = 16
  switch (placement) {
    case 'bottom':
      return { left: rect.left + rect.width / 2 - 160, top: rect.bottom + gap }
    case 'top':
      return { left: rect.left + rect.width / 2 - 160, bottom: window.innerHeight - rect.top + gap }
    case 'right':
      return { left: rect.right + gap, top: rect.top + rect.height / 2 - 80 }
    case 'left':
      return { right: window.innerWidth - rect.left + gap, top: rect.top + rect.height / 2 - 80 }
  }
}
