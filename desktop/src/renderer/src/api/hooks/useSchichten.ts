/**
 * TanStack Query hooks for the Schichten (shift planning) module.
 *
 * Queries for shifts, assignments, templates, compliance checks, and stats.
 * Mutations for shift lifecycle, publish, employee assignment,
 * template management, and template application.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listShifts,
  getShift,
  createShift,
  updateShift,
  deleteShift,
  publishShifts,
  listAssignments,
  assignEmployee,
  unassignEmployee,
  listTemplates,
  createTemplate,
  updateTemplate,
  deleteTemplate,
  applyTemplate,
  checkArbzgCompliance,
  getShiftStats,
} from '../schichten-client'
import type {
  CreateShiftInput,
  UpdateShiftInput,
  ListShiftsParams,
  PublishShiftsInput,
  AssignEmployeeInput,
  CreateTemplateInput,
  UpdateTemplateInput,
  ListTemplatesParams,
  ApplyTemplateInput,
  ArbzgCheckParams,
} from '../schichten-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const schichtenKeys = {
  all: ['schichten'] as const,
  shifts: (params?: ListShiftsParams) => ['schichten', 'shifts', params] as const,
  shift: (id: string) => ['schichten', 'shift', id] as const,
  assignments: (shiftId: string) => ['schichten', 'assignments', shiftId] as const,
  templates: (params?: ListTemplatesParams) => ['schichten', 'templates', params] as const,
  compliance: (params: ArbzgCheckParams) => ['schichten', 'compliance', params] as const,
  stats: (params?: { from?: string; to?: string }) => ['schichten', 'stats', params] as const,
}

const STALE_TIME = 30_000 // 30 s — consistent with other modules

// ---------------------------------------------------------------------------
// Queries — Shifts
// ---------------------------------------------------------------------------

export function useShiftsList(params?: ListShiftsParams) {
  return useQuery({
    queryKey: schichtenKeys.shifts(params),
    queryFn: () => listShifts(params),
    staleTime: STALE_TIME,
  })
}

export function useShift(id: string) {
  return useQuery({
    queryKey: schichtenKeys.shift(id),
    queryFn: () => getShift(id),
    enabled: !!id,
    staleTime: STALE_TIME,
  })
}

// ---------------------------------------------------------------------------
// Queries — Assignments
// ---------------------------------------------------------------------------

export function useShiftAssignments(shiftId: string) {
  return useQuery({
    queryKey: schichtenKeys.assignments(shiftId),
    queryFn: () => listAssignments(shiftId),
    enabled: !!shiftId,
    staleTime: STALE_TIME,
  })
}

// ---------------------------------------------------------------------------
// Queries — Templates
// ---------------------------------------------------------------------------

export function useTemplatesList(params?: ListTemplatesParams) {
  return useQuery({
    queryKey: schichtenKeys.templates(params),
    queryFn: () => listTemplates(params),
    staleTime: STALE_TIME,
  })
}

// ---------------------------------------------------------------------------
// Queries — Compliance & Stats
// ---------------------------------------------------------------------------

export function useArbzgCheck(params: ArbzgCheckParams) {
  return useQuery({
    queryKey: schichtenKeys.compliance(params),
    queryFn: () => checkArbzgCompliance(params),
    enabled: !!(params.employee_id && params.new_shift_start && params.new_shift_end),
    // Bug #20: compliance check must always reflect the latest assignments — never serve stale result
    staleTime: 0,
  })
}

export function useShiftStats(params?: { from?: string; to?: string }) {
  return useQuery({
    queryKey: schichtenKeys.stats(params),
    queryFn: () => getShiftStats(params),
    staleTime: STALE_TIME,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Shifts
// ---------------------------------------------------------------------------

export function useCreateShift() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateShiftInput) => createShift(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['schichten', 'shifts'] })
      qc.invalidateQueries({ queryKey: ['schichten', 'stats'] })
    },
  })
}

export function useUpdateShift() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateShiftInput & { id: string }) =>
      updateShift(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: schichtenKeys.shift(variables.id) })
      qc.invalidateQueries({ queryKey: ['schichten', 'shifts'] })
    },
  })
}

export function useDeleteShift() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteShift(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['schichten', 'shifts'] })
      qc.invalidateQueries({ queryKey: ['schichten', 'stats'] })
    },
  })
}

export function usePublishShifts() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: PublishShiftsInput) => publishShifts(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['schichten', 'shifts'] })
      qc.invalidateQueries({ queryKey: ['schichten', 'stats'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Assignments
// ---------------------------------------------------------------------------

export function useAssignEmployee() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ shiftId, ...body }: AssignEmployeeInput & { shiftId: string }) =>
      assignEmployee(shiftId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: schichtenKeys.assignments(variables.shiftId) })
      qc.invalidateQueries({ queryKey: ['schichten', 'stats'] })
      // Bug #20: invalidate compliance cache after assignment so subsequent checks are fresh
      qc.invalidateQueries({ queryKey: ['schichten', 'compliance'] })
    },
  })
}

export function useUnassignEmployee() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ shiftId, employeeId }: { shiftId: string; employeeId: string }) =>
      unassignEmployee(shiftId, employeeId),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: schichtenKeys.assignments(variables.shiftId) })
      qc.invalidateQueries({ queryKey: ['schichten', 'stats'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Templates
// ---------------------------------------------------------------------------

export function useCreateTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateTemplateInput) => createTemplate(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['schichten', 'templates'] }),
  })
}

export function useUpdateTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateTemplateInput & { id: string }) =>
      updateTemplate(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['schichten', 'templates'] }),
  })
}

export function useDeleteTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteTemplate(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['schichten', 'templates'] }),
  })
}

export function useApplyTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: ApplyTemplateInput & { id: string }) =>
      applyTemplate(id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['schichten', 'shifts'] })
      qc.invalidateQueries({ queryKey: ['schichten', 'stats'] })
    },
  })
}
