/**
 * TanStack Query hooks for the Vermietung module.
 *
 * Queries for rental objects, rentals, inspections, availability, and calendar.
 * Mutations for object/rental/inspection lifecycle and rental state transitions.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listObjects,
  getObject,
  createObject,
  updateObject,
  deleteObject,
  checkAvailability,
  listRentals,
  getRental,
  createRental,
  updateRental,
  deleteRental,
  startRental,
  endRental,
  listInspections,
  createInspection,
  getInspection,
  updateInspection,
  getRentalCalendar,
} from '../vermietung-client'
import type {
  CreateObjectInput,
  UpdateObjectInput,
  ListObjectsParams,
  CreateRentalInput,
  UpdateRentalInput,
  ListRentalsParams,
  CheckAvailabilityParams,
  CreateInspectionInput,
  UpdateInspectionInput,
  ListInspectionsParams,
  GetRentalCalendarParams,
} from '../vermietung-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const vermietungKeys = {
  all: ['vermietung'] as const,
  objects: (params?: ListObjectsParams) => ['vermietung', 'objects', params] as const,
  object: (id: string) => ['vermietung', 'object', id] as const,
  availability: (objectId: string, params: CheckAvailabilityParams) =>
    ['vermietung', 'object', objectId, 'availability', params] as const,
  rentals: (params?: ListRentalsParams) => ['vermietung', 'rentals', params] as const,
  rental: (id: string) => ['vermietung', 'rental', id] as const,
  inspections: (rentalId: string, params?: ListInspectionsParams) =>
    ['vermietung', 'rental', rentalId, 'inspections', params] as const,
  inspection: (id: string) => ['vermietung', 'inspection', id] as const,
  calendar: (month: number, year: number, objectId?: string) =>
    ['vermietung', 'calendar', month, year, objectId] as const,
}

// ---------------------------------------------------------------------------
// Queries — Objects
// ---------------------------------------------------------------------------

export function useObjectsList(params?: ListObjectsParams) {
  return useQuery({
    queryKey: vermietungKeys.objects(params),
    queryFn: () => listObjects(params),
    staleTime: 5 * 60 * 1000,
  })
}

export function useObject(id: string) {
  return useQuery({
    queryKey: vermietungKeys.object(id),
    queryFn: () => getObject(id),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Availability
// ---------------------------------------------------------------------------

export function useAvailabilityCheck(objectId: string, params: CheckAvailabilityParams) {
  return useQuery({
    queryKey: vermietungKeys.availability(objectId, params),
    queryFn: () => checkAvailability(objectId, params),
    enabled: !!objectId && !!params.start_date && !!params.end_date,
    staleTime: 60 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Rentals
// ---------------------------------------------------------------------------

export function useRentalsList(params?: ListRentalsParams) {
  return useQuery({
    queryKey: vermietungKeys.rentals(params),
    queryFn: () => listRentals(params),
    staleTime: 5 * 60 * 1000,
  })
}

export function useRental(id: string) {
  return useQuery({
    queryKey: vermietungKeys.rental(id),
    queryFn: () => getRental(id),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Inspections
// ---------------------------------------------------------------------------

export function useInspectionsList(rentalId: string, params?: ListInspectionsParams) {
  return useQuery({
    queryKey: vermietungKeys.inspections(rentalId, params),
    queryFn: () => listInspections(rentalId, params),
    enabled: !!rentalId,
    staleTime: 5 * 60 * 1000,
  })
}

export function useInspection(id: string) {
  return useQuery({
    queryKey: vermietungKeys.inspection(id),
    queryFn: () => getInspection(id),
    enabled: !!id,
    staleTime: 5 * 60 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Queries — Calendar
// ---------------------------------------------------------------------------

export function useRentalCalendar(params: GetRentalCalendarParams) {
  return useQuery({
    queryKey: vermietungKeys.calendar(params.month, params.year, params.object_id),
    queryFn: () => getRentalCalendar(params),
    enabled: !!params.month && !!params.year,
    staleTime: 60 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Objects
// ---------------------------------------------------------------------------

export function useCreateObject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateObjectInput) => createObject(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vermietung', 'objects'] }),
  })
}

export function useUpdateObject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateObjectInput & { id: string }) =>
      updateObject(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: vermietungKeys.object(variables.id) })
      qc.invalidateQueries({ queryKey: ['vermietung', 'objects'] })
    },
  })
}

export function useDeleteObject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteObject(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['vermietung', 'objects'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Rentals
// ---------------------------------------------------------------------------

export function useCreateRental() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateRentalInput) => createRental(input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['vermietung', 'rentals'] })
      qc.invalidateQueries({ queryKey: ['vermietung', 'calendar'] })
    },
  })
}

export function useUpdateRental() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateRentalInput & { id: string }) =>
      updateRental(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: vermietungKeys.rental(variables.id) })
      qc.invalidateQueries({ queryKey: ['vermietung', 'rentals'] })
      qc.invalidateQueries({ queryKey: ['vermietung', 'calendar'] })
    },
  })
}

export function useDeleteRental() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteRental(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['vermietung', 'rentals'] })
      qc.invalidateQueries({ queryKey: ['vermietung', 'calendar'] })
    },
  })
}

export function useStartRental() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => startRental(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: vermietungKeys.rental(id) })
      qc.invalidateQueries({ queryKey: ['vermietung', 'rentals'] })
      qc.invalidateQueries({ queryKey: ['vermietung', 'calendar'] })
    },
  })
}

export function useEndRental() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => endRental(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: vermietungKeys.rental(id) })
      qc.invalidateQueries({ queryKey: ['vermietung', 'rentals'] })
      qc.invalidateQueries({ queryKey: ['vermietung', 'calendar'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Inspections
// ---------------------------------------------------------------------------

export function useCreateInspection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ rentalId, ...body }: CreateInspectionInput & { rentalId: string }) =>
      createInspection(rentalId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: vermietungKeys.inspections(variables.rentalId) })
    },
  })
}

export function useUpdateInspection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateInspectionInput & { id: string }) =>
      updateInspection(id, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: vermietungKeys.inspection(variables.id) })
    },
  })
}

export function useUploadInspectionPhoto() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, photo_urls }: { id: string; photo_urls: string[] }) =>
      updateInspection(id, { photo_urls }),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: vermietungKeys.inspection(variables.id) })
    },
  })
}
