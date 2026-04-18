/**
 * TanStack Query hooks for the Berichte (Reports/BI) module.
 *
 * Queries for definitions, schedules and dashboard KPIs.
 * Mutations for definition CRUD, report run/export/cache invalidation,
 * and the full schedule lifecycle including toggle.
 *
 * Export downloads trigger a browser download via a hidden <a> element —
 * same pattern as useFinance.ts:downloadBlob.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createDefinition,
  createSchedule,
  deleteDefinition,
  deleteSchedule,
  exportReport,
  getDashboardKPIs,
  getDefinition,
  invalidateCache,
  listDefinitions,
  listSchedules,
  runReport,
  toggleSchedule,
  updateDefinition,
  updateSchedule,
} from '../berichte-client'
import type {
  CreateDefinitionInput,
  CreateScheduleInput,
  ExportReportInput,
  ListDefinitionsParams,
  ListSchedulesParams,
  RunReportInput,
  UpdateDefinitionInput,
  UpdateScheduleInput,
} from '../berichte-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const berichteKeys = {
  all: ['berichte'] as const,
  definitions: (params?: ListDefinitionsParams) =>
    ['berichte', 'definitions', params] as const,
  definition: (id: string) => ['berichte', 'definitions', id] as const,
  schedules: (params?: ListSchedulesParams) =>
    ['berichte', 'schedules', params] as const,
  kpis: (modules?: string[]) =>
    ['berichte', 'kpis', modules?.slice().sort()] as const,
}

// ---------------------------------------------------------------------------
// Blob download helper (identical pattern to useFinance.ts:461)
// ---------------------------------------------------------------------------

function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// ---------------------------------------------------------------------------
// Queries — Definitions
// ---------------------------------------------------------------------------

export function useDefinitions(params?: ListDefinitionsParams) {
  return useQuery({
    queryKey: berichteKeys.definitions(params),
    queryFn: () => listDefinitions(params),
  })
}

export function useDefinition(id: string) {
  return useQuery({
    queryKey: berichteKeys.definition(id),
    queryFn: () => getDefinition(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Schedules
// ---------------------------------------------------------------------------

export function useSchedules(params?: ListSchedulesParams) {
  return useQuery({
    queryKey: berichteKeys.schedules(params),
    queryFn: () => listSchedules(params),
  })
}

// ---------------------------------------------------------------------------
// Queries — Dashboard KPIs
// ---------------------------------------------------------------------------

export function useDashboardKPIs(modules?: string[]) {
  return useQuery({
    queryKey: berichteKeys.kpis(modules),
    queryFn: () => getDashboardKPIs(modules),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Definitions
// ---------------------------------------------------------------------------

export function useCreateDefinition() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateDefinitionInput) => createDefinition(input),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ['berichte', 'definitions'] }),
  })
}

export function useUpdateDefinition() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateDefinitionInput & { id: string }) =>
      updateDefinition(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: berichteKeys.definition(variables.id) })
      qc.invalidateQueries({ queryKey: ['berichte', 'definitions'] })
    },
  })
}

export function useDeleteDefinition() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteDefinition(id),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ['berichte', 'definitions'] }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Run / Cache / Export
// ---------------------------------------------------------------------------

export function useRunReport() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      definitionId,
      ...input
    }: RunReportInput & { definitionId: string }) =>
      runReport(definitionId, input),
    onSuccess: (_data, variables) => {
      if (variables.force_refresh) {
        qc.invalidateQueries({
          queryKey: ['berichte', 'definitions', variables.definitionId],
        })
      }
    },
  })
}

export function useExportReport() {
  return useMutation({
    mutationFn: ({
      definitionId,
      ...input
    }: ExportReportInput & { definitionId: string }) =>
      exportReport(definitionId, input),
    onSuccess: (result) => {
      downloadBlob(result.blob, result.filename)
    },
  })
}

export function useInvalidateCache() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (definitionId: string) => invalidateCache(definitionId),
    onSuccess: (_data, definitionId) => {
      qc.invalidateQueries({
        queryKey: ['berichte', 'definitions', definitionId],
      })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Schedules
// ---------------------------------------------------------------------------

export function useCreateSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateScheduleInput) => createSchedule(input),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ['berichte', 'schedules'] }),
  })
}

export function useUpdateSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateScheduleInput & { id: string }) =>
      updateSchedule(id, input),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ['berichte', 'schedules'] }),
  })
}

export function useDeleteSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteSchedule(id),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ['berichte', 'schedules'] }),
  })
}

export function useToggleSchedule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, active }: { id: string; active: boolean }) =>
      toggleSchedule(id, active),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: ['berichte', 'schedules'] }),
  })
}
