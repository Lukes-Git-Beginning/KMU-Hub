import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Bot, User, LayoutGrid, Compass, Clock, PartyPopper,
  ChevronRight, ChevronLeft,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { cn } from '@/lib/cn'
import { useUIStore } from '@/stores/ui'

interface OnboardingStep {
  id: string
  icon: LucideIcon
  iconColor: string
  iconBg: string
  title: string
  description: string
  features?: string[]
}

type TFunction = (key: string) => string

function buildSteps(t: TFunction): OnboardingStep[] {
  return [
    {
      id: 'welcome',
      icon: Bot,
      iconColor: 'text-primary',
      iconBg: 'bg-primary/10',
      title: t('onboarding.welcome.title'),
      description: t('onboarding.welcome.description'),
    },
    {
      id: 'profile',
      icon: User,
      iconColor: 'text-blue-600 dark:text-blue-400',
      iconBg: 'bg-blue-500/10',
      title: t('onboarding.profile.title'),
      description: t('onboarding.profile.description'),
      features: [
        t('onboarding.profile.feature1'),
        t('onboarding.profile.feature2'),
        t('onboarding.profile.feature3'),
      ],
    },
    {
      id: 'modules',
      icon: LayoutGrid,
      iconColor: 'text-emerald-600 dark:text-emerald-400',
      iconBg: 'bg-emerald-500/10',
      title: t('onboarding.modules.title'),
      description: t('onboarding.modules.description'),
      features: [
        t('onboarding.modules.feature1'),
        t('onboarding.modules.feature2'),
        t('onboarding.modules.feature3'),
        t('onboarding.modules.feature4'),
        t('onboarding.modules.feature5'),
      ],
    },
    {
      id: 'navigation',
      icon: Compass,
      iconColor: 'text-purple-600 dark:text-purple-400',
      iconBg: 'bg-purple-500/10',
      title: t('onboarding.navigation.title'),
      description: t('onboarding.navigation.description'),
      features: [
        t('onboarding.navigation.feature1'),
        t('onboarding.navigation.feature2'),
        t('onboarding.navigation.feature3'),
        t('onboarding.navigation.feature4'),
      ],
    },
    {
      id: 'timetracking',
      icon: Clock,
      iconColor: 'text-amber-600 dark:text-amber-400',
      iconBg: 'bg-amber-500/10',
      title: t('onboarding.timeTracking.title'),
      description: t('onboarding.timeTracking.description'),
      features: [
        t('onboarding.timeTracking.feature1'),
        t('onboarding.timeTracking.feature2'),
        t('onboarding.timeTracking.feature3'),
        t('onboarding.timeTracking.feature4'),
      ],
    },
    {
      id: 'done',
      icon: PartyPopper,
      iconColor: 'text-primary',
      iconBg: 'bg-primary/10',
      title: t('onboarding.done.title'),
      description: t('onboarding.done.description'),
    },
  ]
}

export function OnboardingWizard() {
  const { t } = useTranslation()
  const [currentStep, setCurrentStep] = useState(0)
  const [showConfetti, setShowConfetti] = useState(false)
  const setOnboardingCompleted = useUIStore((s) => s.setOnboardingCompleted)

  const STEPS = buildSteps(t)
  const step = STEPS[currentStep]
  const isFirst = currentStep === 0
  const isLast = currentStep === STEPS.length - 1

  const handleNext = () => {
    if (isLast) {
      setShowConfetti(true)
      setTimeout(() => {
        setOnboardingCompleted(true)
      }, 1500)
    } else {
      setCurrentStep((s) => s + 1)
    }
  }

  const handleSkip = () => {
    setOnboardingCompleted(true)
  }

  const Icon = step.icon

  return (
    <Dialog open>
      <DialogContent className="gap-0 p-0 max-w-lg [&>button:last-child]:hidden">
        <DialogTitle className="sr-only">{t('onboarding.dialogTitle')}</DialogTitle>
        {/* Confetti */}
        {showConfetti && <Confetti />}

        <div className="p-8">
          {/* Step Icon */}
          <div className="flex justify-center mb-6">
            <div className={cn('flex h-16 w-16 items-center justify-center rounded-2xl', step.iconBg)}>
              <Icon className={cn('h-8 w-8', step.iconColor)} />
            </div>
          </div>

          {/* Content */}
          <div className="text-center mb-6">
            <h2 className="text-xl font-bold text-foreground mb-2">{step.title}</h2>
            <p className="text-sm text-muted-foreground">{step.description}</p>
          </div>

          {/* Features */}
          {step.features && (
            <ul className="space-y-2 mb-6">
              {step.features.map((feature, i) => (
                <li key={i} className="flex items-start gap-2 text-sm">
                  <span className="h-1.5 w-1.5 rounded-full bg-primary mt-1.5 shrink-0" />
                  <span className="text-foreground">{feature}</span>
                </li>
              ))}
            </ul>
          )}

          {/* Progress Dots */}
          <div className="flex items-center justify-center gap-2 mb-6">
            {STEPS.map((_, i) => (
              <button
                key={i}
                onClick={() => setCurrentStep(i)}
                className={cn(
                  'h-2 rounded-full transition-all',
                  i === currentStep
                    ? 'w-6 bg-primary'
                    : i < currentStep
                      ? 'w-2 bg-primary/40'
                      : 'w-2 bg-muted-foreground/20',
                )}
              />
            ))}
          </div>

          {/* Buttons */}
          <div className="flex items-center justify-between">
            <div>
              {!isFirst && (
                <Button variant="ghost" size="sm" onClick={() => setCurrentStep((s) => s - 1)}>
                  <ChevronLeft className="h-4 w-4 mr-1" />
                  {t('common.back')}
                </Button>
              )}
            </div>

            <Button onClick={handleNext} disabled={showConfetti}>
              {isLast ? t('onboarding.nav.letsGo') : t('common.next')}
              {!isLast && <ChevronRight className="h-4 w-4 ml-1" />}
            </Button>
          </div>

          {/* Skip Link */}
          {!isLast && !showConfetti && (
            <div className="text-center mt-4">
              <button
                onClick={handleSkip}
                className="text-xs text-muted-foreground hover:text-foreground transition-colors"
              >
                {t('onboarding.nav.skip')}
              </button>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

/** Simple CSS confetti animation */
const CONFETTI_COLORS = ['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#ef4444']

function Confetti() {
  /* eslint-disable react-hooks/purity -- random values generated once on mount via useMemo */
  const particles = useMemo(() => Array.from({ length: 40 }, (_, i) => ({
    left: Math.random() * 100,
    delay: Math.random() * 0.5,
    duration: 1.5 + Math.random() * 1,
    size: 6 + Math.random() * 6,
    color: CONFETTI_COLORS[i % CONFETTI_COLORS.length],
    rotation: Math.random() * 360,
    borderRadius: Math.random() > 0.5 ? '50%' : '2px',
  })), [])
  /* eslint-enable react-hooks/purity */
  return (
    <div className="fixed inset-0 z-[201] pointer-events-none overflow-hidden">
      {particles.map((p, i) => (
          <div
            key={i}
            className="absolute animate-confetti"
            style={{
              left: `${p.left}%`,
              top: '-10px',
              width: p.size,
              height: p.size,
              backgroundColor: p.color,
              borderRadius: p.borderRadius,
              transform: `rotate(${p.rotation}deg)`,
              animationDelay: `${p.delay}s`,
              animationDuration: `${p.duration}s`,
            }}
          />
        ))}
      <style>{`
        @keyframes confetti-fall {
          0% { transform: translateY(0) rotate(0deg); opacity: 1; }
          100% { transform: translateY(100vh) rotate(720deg); opacity: 0; }
        }
        .animate-confetti {
          animation: confetti-fall linear forwards;
        }
      `}</style>
    </div>
  )
}
