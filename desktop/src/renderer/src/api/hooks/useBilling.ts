/**
 * useBilling.ts — TanStack Query Mock-Layer für Billing-Insights (Sprint 1).
 *
 * Kein Backend-Call. Alle Daten aus Mock-Datei oder localStorage.
 * staleTime: Infinity — Daten ändern sich nur durch explizite Mutations.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { useNotificationsStore } from '@/stores/notifications'
import {
  DEFAULT_INSIGHT_SETTINGS,
  formatEUR,
  type InsightSettings,
  type ModuleUsageStats,
  type UserModuleGrant,
  type SupportTier,
} from '@/lib/pricing'
import { calculateRecommendations } from '@/lib/billing-insights'
import {
  MOCK_GRANTS,
  MOCK_INVOICES,
  generateMockUsage,
  type MockInvoice,
} from '@/mocks/billing-mock'

// ───────────────────────── Query Keys ─────────────────────────

const BILLING_KEYS = {
  tenant: ['billing', 'tenant'] as const,
  moduleAssignments: ['billing', 'moduleAssignments'] as const,
  moduleUsage: (days: number) => ['billing', 'moduleUsage', days] as const,
  insightSettings: ['billing', 'insightSettings'] as const,
  recommendations: ['billing', 'recommendations'] as const,
  dismissedHashes: ['billing', 'dismissedHashes'] as const,
  invoiceHistory: ['billing', 'invoiceHistory'] as const,
}

// ───────────────────────── localStorage helpers ─────────────────────────

const LS_INSIGHT_KEY = 'cosmi:billing:insight-settings'
const LS_DISMISSED_KEY = 'cosmi:billing:dismissed-recs'

function readInsightSettings(): InsightSettings {
  try {
    const raw = localStorage.getItem(LS_INSIGHT_KEY)
    if (!raw) return DEFAULT_INSIGHT_SETTINGS
    return { ...DEFAULT_INSIGHT_SETTINGS, ...JSON.parse(raw) }
  } catch {
    return DEFAULT_INSIGHT_SETTINGS
  }
}

function writeInsightSettings(settings: InsightSettings): void {
  localStorage.setItem(LS_INSIGHT_KEY, JSON.stringify(settings))
}

function readDismissedHashes(): Set<string> {
  try {
    const raw = localStorage.getItem(LS_DISMISSED_KEY)
    if (!raw) return new Set()
    return new Set(JSON.parse(raw) as string[])
  } catch {
    return new Set()
  }
}

function writeDismissedHashes(hashes: Set<string>): void {
  localStorage.setItem(LS_DISMISSED_KEY, JSON.stringify([...hashes]))
}

// ───────────────────────── Mock Tenant Type ─────────────────────────

export interface MockTenantData {
  planType: 'cosmi' | 'orbit'
  supportTier: SupportTier
  billingPeriodEnd: string
  totalSeats: number
  status: 'active' | 'past_due' | 'paused'
}

// ───────────────────────── Hooks ─────────────────────────

/** Tenant-Grunddaten (Plan, Support-Tier, Abrechnungszeitraum). */
export function useTenant() {
  return useQuery<MockTenantData>({
    queryKey: BILLING_KEYS.tenant,
    queryFn: () =>
      Promise.resolve({
        planType: 'cosmi',
        supportTier: 'priority' as SupportTier,
        billingPeriodEnd: '2026-04-30',
        totalSeats: 14,
        status: 'active',
      }),
    staleTime: Infinity,
  })
}

/** Alle UserModuleGrant-Einträge (wer hat welches Modul zugeteilt bekommen). */
export function useModuleAssignments() {
  return useQuery<UserModuleGrant[]>({
    queryKey: BILLING_KEYS.moduleAssignments,
    queryFn: () => Promise.resolve(MOCK_GRANTS),
    staleTime: Infinity,
  })
}

/** Berechnete Nutzungsstatistiken per Modul. */
export function useModuleUsage() {
  const insightSettings = useInsightSettings()
  const days = insightSettings.data?.observationDays ?? DEFAULT_INSIGHT_SETTINGS.observationDays

  return useQuery<ModuleUsageStats[]>({
    queryKey: BILLING_KEYS.moduleUsage(days),
    queryFn: () => Promise.resolve(generateMockUsage(MOCK_GRANTS, days)),
    staleTime: Infinity,
    enabled: insightSettings.isSuccess,
  })
}

/** Insight-Einstellungen aus localStorage (mit Default-Fallback). */
export function useInsightSettings() {
  return useQuery<InsightSettings>({
    queryKey: BILLING_KEYS.insightSettings,
    queryFn: () => Promise.resolve(readInsightSettings()),
    staleTime: Infinity,
  })
}

/** Mutation: Insight-Einstellungen schreiben + Cache invalidieren. */
export function useUpdateInsightSettings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (settings: InsightSettings) => {
      writeInsightSettings(settings)
      return settings
    },
    onSuccess: (settings) => {
      queryClient.setQueryData(BILLING_KEYS.insightSettings, settings)
      // Usage-Daten ebenfalls invalidieren (observationDays könnte sich geändert haben)
      void queryClient.invalidateQueries({ queryKey: ['billing', 'moduleUsage'] })
      void queryClient.invalidateQueries({ queryKey: BILLING_KEYS.recommendations })
    },
  })
}

/** Verworfene Recommendation-Hashes aus localStorage. */
export function useDismissedRecommendations() {
  return useQuery<Set<string>>({
    queryKey: BILLING_KEYS.dismissedHashes,
    queryFn: () => Promise.resolve(readDismissedHashes()),
    staleTime: Infinity,
  })
}

/** Mutation: Hash zu Dismissed-Set hinzufügen. */
export function useDismissRecommendation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (hash: string) => {
      const current = readDismissedHashes()
      current.add(hash)
      writeDismissedHashes(current)
      return current
    },
    onSuccess: (hashes) => {
      queryClient.setQueryData(BILLING_KEYS.dismissedHashes, hashes)
      void queryClient.invalidateQueries({ queryKey: BILLING_KEYS.recommendations })
    },
  })
}

/** Kombinierter Hook: berechnet Recommendations aus Usage + Settings + Dismissed. */
export function useRecommendations() {
  const usage = useModuleUsage()
  const settings = useInsightSettings()
  const dismissed = useDismissedRecommendations()

  return useQuery({
    queryKey: BILLING_KEYS.recommendations,
    queryFn: () => {
      const usageData = usage.data ?? []
      const settingsData = settings.data ?? DEFAULT_INSIGHT_SETTINGS
      const dismissedData = dismissed.data ?? new Set<string>()
      return Promise.resolve(calculateRecommendations(usageData, settingsData, dismissedData))
    },
    staleTime: Infinity,
    enabled: usage.isSuccess && settings.isSuccess && dismissed.isSuccess,
  })
}

// ───────────────────────── Notification-Layer ─────────────────────────

/**
 * Side-Effect-Hook: erzeugt Bell-Notifications wenn neue Recommendations erscheinen.
 * Vergleicht via Hash-Set ob Recommendations tatsächlich neu sind (Render-stabil via useRef).
 *
 * Aufruf: einmalig in BillingSettingsTab (oder AppShell falls globale Überwachung gewünscht).
 * Kein Return-Value — reiner Seiteneffekt.
 */
export function useBillingRecommendationNotifier() {
  const recommendations = useRecommendations()
  const settings = useInsightSettings()
  const addNotification = useNotificationsStore((s) => s.addNotification)
  const lastSeenHashes = useRef<Set<string>>(new Set())

  useEffect(() => {
    const recs = recommendations.data
    const enabled = settings.data?.recommendationsEnabled ?? true
    if (!recs || !enabled || recs.length === 0) return

    const newRecs = recs.filter((r) => !lastSeenHashes.current.has(r.hash))
    if (newRecs.length === 0) return

    const totalSavings = newRecs.reduce((sum, r) => sum + r.monthlySavings, 0)

    addNotification({
      type: 'billing_recommendation',
      title: 'Neue Cost-Saving-Empfehlung',
      body:
        newRecs.length === 1
          ? `1 neue Empfehlung mit ${formatEUR(totalSavings)} Sparpotential`
          : `${newRecs.length} neue Empfehlungen mit ${formatEUR(totalSavings)} Sparpotential`,
      icon: 'lightbulb',
      priority: 'low',
      actionUrl: '/admin/billing',
    })

    newRecs.forEach((r) => lastSeenHashes.current.add(r.hash))
  }, [recommendations.data, settings.data?.recommendationsEnabled, addNotification])
}

// ───────────────────────── Cost-Savings Tracker ─────────────────────────

const LS_SAVINGS_KEY = 'cosmi:billing:cost-savings'

interface SavingsEntry {
  moduleId: string
  savedAt: string
  monthlySavings: number
}

interface SavingsData {
  totalMonthly: number
  applyCount: number
  history: SavingsEntry[]
}

const SAVINGS_KEYS = {
  savings: ['billing', 'costSavings'] as const,
}

function readSavingsData(): SavingsData {
  try {
    const raw = localStorage.getItem(LS_SAVINGS_KEY)
    if (!raw) return { totalMonthly: 0, applyCount: 0, history: [] }
    return JSON.parse(raw) as SavingsData
  } catch {
    return { totalMonthly: 0, applyCount: 0, history: [] }
  }
}

function writeSavingsData(data: SavingsData): void {
  localStorage.setItem(LS_SAVINGS_KEY, JSON.stringify(data))
}

export interface CostSavingsResult {
  totalSaved: number
  applyCount: number
}

/** Liest den kumulierten Cost-Saving-Counter aus localStorage. */
export function useCostSavingsTracker() {
  return useQuery<CostSavingsResult>({
    queryKey: SAVINGS_KEYS.savings,
    queryFn: () => {
      const data = readSavingsData()
      return Promise.resolve({ totalSaved: data.totalMonthly, applyCount: data.applyCount })
    },
    staleTime: Infinity,
  })
}

export interface TrackSavingsArgs {
  moduleId: string
  monthlySavings: number
}

/**
 * Mutation: nach erfolgreichem Apply einer Recommendation aufrufen.
 * Persistiert in localStorage und aktualisiert den Query-Cache.
 * Zeigt einen Joy-Toast wenn settings.toastOnApply === true.
 */
export function useTrackSavings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (args: TrackSavingsArgs): Promise<SavingsData> => {
      const current = readSavingsData()
      const next: SavingsData = {
        totalMonthly: current.totalMonthly + args.monthlySavings,
        applyCount: current.applyCount + 1,
        history: [
          { moduleId: args.moduleId, savedAt: new Date().toISOString(), monthlySavings: args.monthlySavings },
          ...current.history,
        ],
      }
      writeSavingsData(next)
      return next
    },
    onSuccess: (data) => {
      queryClient.setQueryData<CostSavingsResult>(SAVINGS_KEYS.savings, {
        totalSaved: data.totalMonthly,
        applyCount: data.applyCount,
      })
    },
  })
}

/** Cosmi-Rechnungs-Historie (Mock). */
export function useInvoiceHistory() {
  return useQuery<MockInvoice[]>({
    queryKey: BILLING_KEYS.invoiceHistory,
    queryFn: () => Promise.resolve(MOCK_INVOICES),
    staleTime: Infinity,
  })
}
