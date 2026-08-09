/**
 * AnpassungenTab — the "Anpassungen" tab in AdminHubPage: the module-editor hub.
 *
 * E-4 module gallery: one card per editable module (fixed module name + what is
 * customizable: Begriffe / Wertelisten / Felder counts + a live status badge).
 * E-5b rollouts list: saved drafts + scheduled rollouts + live deploys with
 * 1-click rollback. Both read the drafts store via a React Query keyed on
 * 'customization', which the cross-window sync listener invalidates when a deploy
 * lands from the editor window (mock; Luke's shared DB replaces the mirror).
 *
 * Customization happens ONLY inside the module editor (Darien 2026-07-22) — this
 * hub launches it and manages the change-management around it.
 *
 * Gated on `admin:customization:manage` (admin + it_admin) via AdminHubPage.
 */
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Contact, LifeBuoy, Wand2, RotateCcw, Trash2, PencilLine, Clock, CircleDot } from 'lucide-react'
import { ModuleLoadingFallback } from '@/components/layout/ModuleShell'
import { useCapabilitySet } from '@/hooks/useCapability'
import { i18n } from '@/i18n/i18n'
import { applyLabelOverlay } from '@/i18n/useLabelOverlay'
import { resolveLabelOverrides } from '@/mocks/data/customization'
import { listCustomFields } from '@/mocks/data/custom-fields'
import {
  listDrafts,
  getDraft,
  rollbackDeploy,
  canRollback,
  deleteDraft,
  promoteDraftById,
  setDraftSchedule,
  clearDraftSchedule,
  setDraftAnnouncement,
  renameDraft,
  getDeploySnapshot,
} from '@/mocks/data/customization-drafts'
import type { CustomizationDraft, DraftStatus } from '@/api/customization-types'
import { EDITOR_MODULES, type EditorModuleDef } from './editor/editorModules'
import {
  stashDraftForEditor,
  clearStashedDraft,
  publishDraftMirror,
  publishCustomizationDeploy,
} from './editor/customization-sync'
import { RolloutDetailModal } from './RolloutDetailModal'

/** Lucide icon per editor-module (matches EditorModuleDef.icon). */
const MODULE_ICON = { contact: Contact, lifeBuoy: LifeBuoy } as const

const STATUS_STYLE: Record<DraftStatus, string> = {
  draft: 'bg-muted text-muted-foreground',
  scheduled: 'bg-amber-500/10 text-amber-600 dark:text-amber-400',
  live: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
  superseded: 'bg-muted text-muted-foreground/70',
}

export default function AnpassungenTab() {
  const { t } = useTranslation()
  const { ready } = useCapabilitySet()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  // Rollout opened in the detail modal (Cosmi convention: row click → modal).
  const [detailId, setDetailId] = useState<string | null>(null)

  const { data: drafts = [] } = useQuery({
    queryKey: ['customization', 'drafts'],
    queryFn: () => listDrafts(),
    // The drafts store is mutated by deploys in the editor window; the sync
    // broadcast can be missed while the hub is unmounted, so always refetch on
    // mount (matches the R-5 audit-log freshness fix).
    staleTime: 0,
    refetchOnMount: 'always',
  })

  const refresh = (): void => {
    void queryClient.invalidateQueries({
      predicate: (q) => JSON.stringify(q.queryKey).toLowerCase().includes('customization'),
    })
  }

  /**
   * Open the editor in its own OS window (web fallback: same-window route).
   *
   * `handedOver` says a draft was just stashed for the window that is about to
   * boot. Only one editor window exists per module, so a click that merely
   * focuses an open one must take that note back — otherwise it sits in storage
   * and the NEXT launch opens a state from two saves ago (Darien 2026-08-09).
   * That window gets the draft through the channel instead.
   */
  const openEditor = (key: string, handedOver = false): void => {
    if (window.electronAPI?.editor) {
      void window.electronAPI.editor.openWindow(key).then((created) => {
        // Only an explicit "no window was created" clears the note. An older main
        // process answers undefined — clearing then would delete the handover out
        // from under the window that is booting to read it.
        if (handedOver && created === false) clearStashedDraft()
      })
    } else {
      navigate(`/editor-window?module=${encodeURIComponent(key)}`)
    }
  }

  /**
   * Opening a module that already has an unfinished draft continues it (Darien
   * 2026-08-05: "wenn man den Entwurf öffnet, ist es wieder das aktuelle
   * Layout"). Saving something and then finding a blank editor reads as data
   * loss — a saved draft is the state of that module until it is deployed or
   * discarded. The editor says which draft it loaded and offers a fresh start.
   */
  const openModule = (key: string): void => {
    // Read the store, not the query cache. The cache is filled by a broadcast from
    // the editor window, and if that has not landed yet the tile would hand over
    // nothing — which is what "manchmal öffnet er es so wie es aktuell ist" was
    // (Darien 2026-08-09). listDrafts() re-reads shared storage every call.
    const pending = listDrafts(key).find((d) => d.status === 'draft')
    if (pending) stashDraftForEditor(pending)
    openEditor(key, Boolean(pending))
  }

  const handleRollback = (draft: CustomizationDraft): void => {
    rollbackDeploy(draft.id)
    // Revert the live overlay in THIS window (sidebar/headings) + refetch data.
    applyLabelOverlay(i18n.language, resolveLabelOverrides(i18n.language))
    refresh()
    toast.success(t('customization.editor.rollouts.rolledBack'))
  }

  const handleDelete = (draft: CustomizationDraft): void => {
    deleteDraft(draft.id)
    setDetailId(null)
    refresh()
    toast.success(t('customization.editor.rollouts.deleted'))
  }

  /**
   * Continue a saved draft. The pencil used to open a BLANK editor — the record
   * existed, its changes did not travel. The editor window has its own JS heap, so
   * the draft is handed over through storage both windows share.
   */
  const continueDraft = (draft: CustomizationDraft): void => {
    // Same reason as openModule: hand over what is stored right now, not the copy
    // this list was rendered from.
    stashDraftForEditor(getDraft(draft.id) ?? draft)
    setDetailId(null)
    openEditor(draft.moduleKey, true)
  }

  const handleDeployNow = (draft: CustomizationDraft): void => {
    const promoted = promoteDraftById(draft.id)
    if (!promoted) return
    // This window IS the one that promoted it, so apply the live label overlay here
    // and mirror the record + rollback snapshot to any other window.
    applyLabelOverlay(i18n.language, resolveLabelOverrides(i18n.language))
    publishCustomizationDeploy(promoted.payload, promoted, getDeploySnapshot(promoted.id))
    setDetailId(null)
    refresh()
    toast.success(t('customization.editor.toast.applied'))
  }

  const handleSchedule = (draft: CustomizationDraft, scheduledAt: string): void => {
    const updated = setDraftSchedule(draft.id, scheduledAt)
    if (!updated) return
    publishDraftMirror(updated)
    refresh()
    toast.success(t('customization.editor.deploy.toastScheduled'))
  }

  const handleUnschedule = (draft: CustomizationDraft): void => {
    const updated = clearDraftSchedule(draft.id)
    if (!updated) return
    publishDraftMirror(updated)
    refresh()
    toast.success(t('customization.rollout.unscheduled'))
  }

  const handleRename = (draft: CustomizationDraft, name: string): void => {
    const updated = renameDraft(draft.id, name)
    if (!updated) return
    publishDraftMirror(updated)
    refresh()
    toast.success(t('customization.editor.toast.renamed', { name: updated.name }))
  }

  const handleAnnouncement = (draft: CustomizationDraft, text: string): void => {
    const updated = setDraftAnnouncement(draft.id, text)
    if (!updated) return
    publishDraftMirror(updated)
    refresh()
  }

  const detail = detailId ? drafts.find((d) => d.id === detailId) : undefined

  const fmtDate = (iso?: string): string =>
    iso ? new Date(iso).toLocaleString(i18n.language, { dateStyle: 'medium', timeStyle: 'short' }) : ''

  if (!ready) return <ModuleLoadingFallback />

  return (
    <div className="flex h-full flex-col overflow-y-auto px-6 py-6">
      {/* ── Header ──────────────────────────────────────────────────────── */}
      <div className="mb-2.5 flex items-center gap-2">
        <div className="flex h-7 w-7 items-center justify-center rounded-md bg-[var(--accent-1)]/10 text-[var(--accent-1)]">
          <Wand2 className="h-4 w-4" aria-hidden="true" />
        </div>
        <h2 className="text-base font-semibold text-foreground">{t('customization.editor.launch.title')}</h2>
        <span className="rounded-full bg-amber-500/15 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-600 dark:text-amber-400">
          {t('customization.editor.launch.beta')}
        </span>
      </div>
      <p className="mb-5 max-w-2xl text-sm leading-relaxed text-muted-foreground">
        {t('customization.editor.launch.subtitle')}
      </p>

      {/* ── E-4 Module gallery ──────────────────────────────────────────── */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {EDITOR_MODULES.map((mod) => (
          <ModuleCard
            key={mod.key}
            mod={mod}
            drafts={drafts.filter((d) => d.moduleKey === mod.key)}
            onOpen={() => openModule(mod.key)}
          />
        ))}
      </div>

      {/* ── E-5b Rollouts & drafts ──────────────────────────────────────── */}
      <div className="mt-8">
        <h3 className="mb-3 text-sm font-semibold text-foreground">{t('customization.editor.rollouts.title')}</h3>
        {drafts.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border px-5 py-8 text-center">
            <Clock className="mx-auto h-6 w-6 text-muted-foreground/25" aria-hidden="true" />
            <p className="mt-2 text-sm text-muted-foreground">{t('customization.editor.rollouts.empty')}</p>
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            {drafts.map((d) => {
              const mod = EDITOR_MODULES.find((m) => m.key === d.moduleKey)
              return (
                // Whole row opens the detail (Cosmi convention) — the icons on the
                // right stay as shortcuts and stop the click from bubbling.
                <div
                  key={d.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => setDetailId(d.id)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault()
                      setDetailId(d.id)
                    }
                  }}
                  className="flex cursor-pointer items-center gap-3 rounded-lg border border-border bg-card px-4 py-2.5 transition-colors hover:border-[var(--accent-1)]/40 hover:bg-accent/30 focus:outline-none focus-visible:ring-2 focus-visible:ring-focus-ring"
                >
                  <CircleDot className={`h-3.5 w-3.5 shrink-0 ${STATUS_STYLE[d.status].split(' ').find((c) => c.startsWith('text-')) ?? ''}`} aria-hidden="true" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium text-foreground">{d.name}</span>
                      <span className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium ${STATUS_STYLE[d.status]}`}>
                        {t(`customization.editor.rollouts.status.${d.status}`)}
                      </span>
                    </div>
                    <p className="truncate text-[11px] text-muted-foreground">
                      {mod ? t(mod.titleKey) : d.moduleKey}
                      {d.status === 'scheduled' && d.scheduledAt
                        ? ` · ${t('customization.editor.rollouts.scheduledFor', { date: fmtDate(d.scheduledAt) })}`
                        : ` · ${fmtDate(d.updatedAt)}`}
                    </p>
                  </div>

                  <div className="flex shrink-0 items-center gap-1" onClick={(e) => e.stopPropagation()}>
                    {d.status === 'live' && canRollback(d.id) && (
                      <button
                        type="button"
                        onClick={() => handleRollback(d)}
                        className="flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                      >
                        <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
                        {t('customization.editor.rollouts.rollback')}
                      </button>
                    )}
                    {d.status === 'draft' && (
                      <button
                        type="button"
                        onClick={() => continueDraft(d)}
                        aria-label={t('customization.editor.rollouts.reopen')}
                        title={t('customization.editor.rollouts.reopen')}
                        className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                      >
                        <PencilLine className="h-3.5 w-3.5" aria-hidden="true" />
                      </button>
                    )}
                    {(d.status === 'draft' || d.status === 'scheduled') && (
                      <button
                        type="button"
                        onClick={() => handleDelete(d)}
                        aria-label={t('common.delete')}
                        title={t('common.delete')}
                        className="flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
                      >
                        <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                      </button>
                    )}
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {detail && (
        <RolloutDetailModal
          draft={detail}
          moduleName={
            EDITOR_MODULES.find((m) => m.key === detail.moduleKey)
              ? t(EDITOR_MODULES.find((m) => m.key === detail.moduleKey)!.titleKey)
              : detail.moduleKey
          }
          canRollback={detail.status === 'live' && canRollback(detail.id)}
          onClose={() => setDetailId(null)}
          onContinue={() => continueDraft(detail)}
          onDeployNow={() => handleDeployNow(detail)}
          onSchedule={(at) => handleSchedule(detail, at)}
          onUnschedule={() => handleUnschedule(detail)}
          onAnnouncementChange={(text) => handleAnnouncement(detail, text)}
          onRename={(name) => handleRename(detail, name)}
          onRollback={() => {
            handleRollback(detail)
            setDetailId(null)
          }}
          onDelete={() => handleDelete(detail)}
        />
      )}
    </div>
  )
}

// ── Module gallery card (E-4) ───────────────────────────────────────────────────

/**
 * What this module actually has right now (Darien 2026-08-05: "die Daten hier
 * sind nur Mocks und ziehen sich das nicht wie es wirklich ist"). The chips used
 * to render the definition's array lengths — so Helpdesk claimed "0 Begriffe"
 * while eight of its headings are renamable, and "1 Feld-Typ" was the number of
 * ENTITIES, not of fields.
 */
function moduleStats(mod: EditorModuleDef): { terms: number; valueSets: number; fields: number } {
  const labels = resolveLabelOverrides(i18n.language)
  const prefix = `${mod.key}.`
  const terms = Object.keys(labels).filter((k) => k.startsWith(prefix) || mod.labelKeys.includes(k))
  const fields = mod.fieldEntities.flatMap((entity) => listCustomFields(entity))
  // A value list a module field points at belongs to that module too, even when
  // it was created later in the editor.
  const bound = fields.map((f) => f.valueSetId).filter((id): id is string => Boolean(id))
  return {
    terms: terms.length,
    valueSets: new Set([...mod.valueSetIds, ...bound]).size,
    fields: fields.length,
  }
}

function ModuleCard({
  mod,
  drafts,
  onOpen,
}: {
  mod: EditorModuleDef
  drafts: CustomizationDraft[]
  onOpen: () => void
}): React.ReactElement {
  const { t } = useTranslation()
  const Icon = MODULE_ICON[mod.icon]
  // Through the query cache so a deploy (which invalidates 'customization')
  // refreshes the numbers instead of leaving stale ones on screen.
  const { data: stats = { terms: 0, valueSets: 0, fields: 0 } } = useQuery({
    queryKey: ['customization', 'module-stats', mod.key],
    queryFn: () => moduleStats(mod),
    staleTime: 0,
    refetchOnMount: 'always',
  })

  // Live status: scheduled rollout > active (live) customization > standard.
  const scheduled = drafts.some((d) => d.status === 'scheduled')
  const live = drafts.some((d) => d.status === 'live')
  const status = scheduled ? 'scheduled' : live ? 'customized' : 'standard'
  const statusStyle =
    status === 'scheduled'
      ? 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
      : status === 'customized'
        ? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
        : 'bg-muted text-muted-foreground'

  return (
    <button
      type="button"
      onClick={onOpen}
      className="group flex flex-col gap-2.5 rounded-xl border bg-background px-4 py-3.5 text-left transition-colors hover:border-[var(--accent-1)]/40 hover:bg-[var(--accent-1)]/5"
    >
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground transition-colors group-hover:bg-[var(--accent-1)]/10 group-hover:text-[var(--accent-1)]">
          <Icon className="h-5 w-5" aria-hidden="true" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-foreground">{t(mod.titleKey)}</p>
          <p className="truncate text-xs text-muted-foreground">{t('customization.editor.launch.open')}</p>
        </div>
        <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium ${statusStyle}`}>
          {t(`customization.editor.status.${status}`)}
        </span>
      </div>
      {/* What is customizable in this module (the manifest, at a glance). */}
      <div className="flex flex-wrap gap-1.5">
        <DimensionChip label={t('customization.editor.gallery.terms', { count: stats.terms })} />
        <DimensionChip label={t('customization.editor.gallery.valueSets', { count: stats.valueSets })} />
        <DimensionChip label={t('customization.editor.gallery.fields', { count: stats.fields })} />
      </div>
    </button>
  )
}

function DimensionChip({ label }: { label: string }): React.ReactElement {
  return (
    <span className="rounded-md bg-muted px-1.5 py-0.5 text-[11px] text-muted-foreground">{label}</span>
  )
}
