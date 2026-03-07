import { useState, useMemo } from 'react'
import {
  Sparkles, User, LayoutGrid, Compass, Clock, PartyPopper,
  ChevronRight, ChevronLeft,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Button } from '@/components/ui/button'
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

const STEPS: OnboardingStep[] = [
  {
    id: 'welcome',
    icon: Sparkles,
    iconColor: 'text-primary',
    iconBg: 'bg-primary/10',
    title: 'Willkommen bei KMU Hub!',
    description: 'Dein All-in-One Arbeitsplatz für dein Unternehmen. Lass uns die wichtigsten Funktionen entdecken.',
  },
  {
    id: 'profile',
    icon: User,
    iconColor: 'text-blue-600 dark:text-blue-400',
    iconBg: 'bg-blue-500/10',
    title: 'Dein Profil',
    description: 'Richte dein Profil ein, damit dein Team dich erkennen kann.',
    features: [
      'Profilbild und persönliche Informationen',
      'Zeiterfassung und Stundenübersicht',
      'Abwesenheiten und Urlaubsanträge',
    ],
  },
  {
    id: 'modules',
    icon: LayoutGrid,
    iconColor: 'text-emerald-600 dark:text-emerald-400',
    iconBg: 'bg-emerald-500/10',
    title: 'Deine Module',
    description: 'KMU Hub vereint alles, was du zum Arbeiten brauchst.',
    features: [
      'Projekte & Aufgaben verwalten',
      'Chat & Meetings mit dem Team',
      'E-Mail & Kalender organisieren',
      'Dokumente & Kontakte pflegen',
      'Finanzen & Team-Verwaltung',
    ],
  },
  {
    id: 'navigation',
    icon: Compass,
    iconColor: 'text-purple-600 dark:text-purple-400',
    iconBg: 'bg-purple-500/10',
    title: 'Navigation',
    description: 'Finde dich schnell in KMU Hub zurecht.',
    features: [
      'Sidebar links für alle Module',
      'Ctrl+K für die globale Suche',
      'Tagesplanung direkt im Header',
      'Schreibtisch-Ansicht mit persönlichem Touch',
    ],
  },
  {
    id: 'timetracking',
    icon: Clock,
    iconColor: 'text-amber-600 dark:text-amber-400',
    iconBg: 'bg-amber-500/10',
    title: 'Zeiterfassung',
    description: 'Tracke deine Arbeitszeit direkt im Header — einfach und schnell.',
    features: [
      'Ein-Klick Timer mit Kategorien',
      'Vorlagen für wiederkehrende Aufgaben',
      'Soll/Ist Vergleich und Überstunden',
      'Team-Übersicht: Wer arbeitet woran?',
    ],
  },
  {
    id: 'done',
    icon: PartyPopper,
    iconColor: 'text-primary',
    iconBg: 'bg-primary/10',
    title: 'Alles bereit!',
    description: 'Du bist startklar. Erkunde KMU Hub und mach dein Team produktiver.',
  },
]

export function OnboardingWizard() {
  const [currentStep, setCurrentStep] = useState(0)
  const [showConfetti, setShowConfetti] = useState(false)
  const setOnboardingCompleted = useUIStore((s) => s.setOnboardingCompleted)

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
    <div className="fixed inset-0 z-[200] flex items-center justify-center bg-black/60 backdrop-blur-sm">
      {/* Confetti */}
      {showConfetti && <Confetti />}

      {/* Card */}
      <div className="relative max-w-lg w-full mx-4 rounded-2xl border border-border bg-card p-8 shadow-2xl">
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
                Zurueck
              </Button>
            )}
          </div>

          <Button onClick={handleNext} disabled={showConfetti}>
            {isLast ? "Los geht's!" : 'Weiter'}
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
              Überspringen
            </button>
          </div>
        )}
      </div>
    </div>
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
