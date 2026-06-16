/**
 * FinanceDetailNav — zentrale Navigation zwischen den Beleg-Detail-Modals
 * (Rechnung ⇄ Angebot ⇄ Gutschrift). Hält einen Navigations-Stack, sodass
 * „Verknüpfungen" (Quell-Angebot, Original-Rechnung, Kundenkonto-Belege …)
 * von Modal zu Modal navigieren — mit Zurück-Verhalten wie eine Mini-360°-Ansicht.
 *
 * Provider lebt in FinanzenPage und rendert das oberste Stack-Element zentral.
 * Recurring-/Expense-/Transaction-Tabs sind Kinder des Providers und können
 * über `useFinanceDetailNavOptional()` ebenfalls Belege öffnen (z.B. eine von
 * einem Abo erzeugte Rechnung).
 */
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'

/** Beleg-Typen, die zentral als Detail-Modal geöffnet werden können. */
export type FinanceDocKind = 'invoice' | 'quote' | 'creditNote'

export interface FinanceDocRef {
  kind: FinanceDocKind
  id: string
}

interface FinanceDetailNavValue {
  /** Aktuell oben liegender Beleg (oder null, wenn kein Modal offen ist). */
  current: FinanceDocRef | null
  /** Tiefe des Navigations-Stacks (1 = Wurzel, >1 = „Zurück" verfügbar). */
  depth: number
  /** Beleg öffnen (auf den Stack legen). */
  open: (kind: FinanceDocKind, id: string) => void
  /** Einen Schritt zurück (oberstes Element entfernen). */
  back: () => void
  /** Gesamten Stack schließen (Modal-Kette beenden). */
  close: () => void
}

const FinanceDetailNavContext = createContext<FinanceDetailNavValue | null>(null)

export function FinanceDetailNavProvider({ children }: { children: ReactNode }) {
  const [stack, setStack] = useState<FinanceDocRef[]>([])

  const open = useCallback((kind: FinanceDocKind, id: string) => {
    setStack((s) => {
      const top = s[s.length - 1]
      // Denselben Beleg nicht doppelt direkt aufeinander legen.
      if (top && top.kind === kind && top.id === id) return s
      return [...s, { kind, id }]
    })
  }, [])

  const back = useCallback(() => setStack((s) => s.slice(0, -1)), [])
  const close = useCallback(() => setStack([]), [])

  const value = useMemo<FinanceDetailNavValue>(
    () => ({ current: stack[stack.length - 1] ?? null, depth: stack.length, open, back, close }),
    [stack, open, back, close],
  )

  return <FinanceDetailNavContext.Provider value={value}>{children}</FinanceDetailNavContext.Provider>
}

/** Innerhalb des Providers: wirft, wenn versehentlich außerhalb genutzt. */
export function useFinanceDetailNav(): FinanceDetailNavValue {
  const ctx = useContext(FinanceDetailNavContext)
  if (!ctx) throw new Error('useFinanceDetailNav must be used within FinanceDetailNavProvider')
  return ctx
}

/** Für Komponenten, die auch ohne Provider rendern können (gibt dann null zurück). */
export function useFinanceDetailNavOptional(): FinanceDetailNavValue | null {
  return useContext(FinanceDetailNavContext)
}
