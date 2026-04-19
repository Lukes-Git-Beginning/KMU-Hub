/**
 * TanStack Query hooks for the Formulare (Forms) module.
 *
 * Queries for schemas, submissions, webhooks, and deliveries.
 * Mutations for schema CRUD + duplicate, submission lifecycle,
 * webhook CRUD, and submission export with browser download trigger.
 */
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createFormSchema,
  createSubmission,
  createWebhook,
  deleteFormSchema,
  deleteWebhook,
  duplicateFormSchema,
  exportSubmissions,
  getFormSchema,
  getSubmission,
  getWebhook,
  listAllDeliveries,
  listFormSchemas,
  listSubmissions,
  listWebhookDeliveries,
  listWebhooks,
  updateFormSchema,
  updateSubmissionStatus,
  updateWebhook,
} from '../formulare-client'
import type {
  CreateFormSchemaInput,
  CreateSubmissionInput,
  CreateWebhookInput,
  DuplicateFormSchemaInput,
  ExportFormat,
  ListDeliveriesQuery,
  ListSchemasQuery,
  ListSubmissionsQuery,
  UpdateFormSchemaInput,
  UpdateSubmissionStatusInput,
  UpdateWebhookInput,
} from '../formulare-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const formulareKeys = {
  all: ['formulare'] as const,
  schemas: {
    all: ['formulare', 'schemas'] as const,
    list: (q?: ListSchemasQuery) => ['formulare', 'schemas', 'list', q] as const,
    detail: (id: string) => ['formulare', 'schemas', 'detail', id] as const,
  },
  submissions: {
    all: ['formulare', 'submissions'] as const,
    listBySchema: (schemaId: string, q?: ListSubmissionsQuery) =>
      ['formulare', 'submissions', 'list', schemaId, q] as const,
    detail: (id: string) => ['formulare', 'submissions', 'detail', id] as const,
  },
  webhooks: {
    all: ['formulare', 'webhooks'] as const,
    listBySchema: (schemaId: string) =>
      ['formulare', 'webhooks', 'list', schemaId] as const,
    detail: (id: string) => ['formulare', 'webhooks', 'detail', id] as const,
  },
  deliveries: {
    byWebhook: (webhookId: string) =>
      ['formulare', 'deliveries', 'webhook', webhookId] as const,
    all: (q?: ListDeliveriesQuery) => ['formulare', 'deliveries', 'all', q] as const,
  },
}

// ---------------------------------------------------------------------------
// Blob download helper (same pattern as useBerichte.ts:downloadBlob)
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
// Queries — Schemas
// ---------------------------------------------------------------------------

export function useFormSchemas(query?: ListSchemasQuery) {
  return useQuery({
    queryKey: formulareKeys.schemas.list(query),
    queryFn: () => listFormSchemas(query),
  })
}

export function useFormSchema(id: string) {
  return useQuery({
    queryKey: formulareKeys.schemas.detail(id),
    queryFn: () => getFormSchema(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Submissions
// ---------------------------------------------------------------------------

export function useSubmissions(formSchemaId: string, query?: ListSubmissionsQuery) {
  return useQuery({
    queryKey: formulareKeys.submissions.listBySchema(formSchemaId, query),
    queryFn: () => listSubmissions(formSchemaId, query),
    enabled: !!formSchemaId,
  })
}

export function useSubmission(id: string) {
  return useQuery({
    queryKey: formulareKeys.submissions.detail(id),
    queryFn: () => getSubmission(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Webhooks
// ---------------------------------------------------------------------------

export function useWebhooks(formSchemaId: string) {
  return useQuery({
    queryKey: formulareKeys.webhooks.listBySchema(formSchemaId),
    queryFn: () => listWebhooks(formSchemaId),
    enabled: !!formSchemaId,
  })
}

export function useWebhook(id: string) {
  return useQuery({
    queryKey: formulareKeys.webhooks.detail(id),
    queryFn: () => getWebhook(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Deliveries
// ---------------------------------------------------------------------------

export function useWebhookDeliveries(webhookId: string) {
  return useQuery({
    queryKey: formulareKeys.deliveries.byWebhook(webhookId),
    queryFn: () => listWebhookDeliveries(webhookId),
    enabled: !!webhookId,
  })
}

export function useAllDeliveries(query?: ListDeliveriesQuery) {
  return useQuery({
    queryKey: formulareKeys.deliveries.all(query),
    queryFn: () => listAllDeliveries(query),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Schemas
// ---------------------------------------------------------------------------

export function useCreateFormSchema() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateFormSchemaInput) => createFormSchema(input),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: formulareKeys.schemas.all }),
  })
}

export function useUpdateFormSchema() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateFormSchemaInput & { id: string }) =>
      updateFormSchema(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: formulareKeys.schemas.detail(variables.id),
      })
      qc.invalidateQueries({ queryKey: formulareKeys.schemas.all })
    },
  })
}

export function useDeleteFormSchema() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteFormSchema(id),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: formulareKeys.schemas.all }),
  })
}

export function useDuplicateFormSchema() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      ...input
    }: DuplicateFormSchemaInput & { id: string }) =>
      duplicateFormSchema(id, input),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: formulareKeys.schemas.all }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Submissions
// ---------------------------------------------------------------------------

export function useCreateSubmission() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      formSchemaId,
      ...input
    }: CreateSubmissionInput & { formSchemaId: string }) =>
      createSubmission(formSchemaId, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: formulareKeys.submissions.listBySchema(variables.formSchemaId),
      })
      // Invalidate schema detail to refresh submissionCount
      qc.invalidateQueries({
        queryKey: formulareKeys.schemas.detail(variables.formSchemaId),
      })
    },
  })
}

export function useUpdateSubmissionStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      formSchemaId: _formSchemaId,
      ...input
    }: UpdateSubmissionStatusInput & { id: string; formSchemaId?: string }) =>
      updateSubmissionStatus(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: formulareKeys.submissions.detail(variables.id),
      })
      if (variables.formSchemaId) {
        qc.invalidateQueries({
          queryKey: formulareKeys.submissions.listBySchema(
            variables.formSchemaId,
          ),
        })
      } else {
        qc.invalidateQueries({ queryKey: formulareKeys.submissions.all })
      }
    },
  })
}

export function useExportSubmissions() {
  return useMutation({
    mutationFn: ({
      formSchemaId,
      format,
    }: {
      formSchemaId: string
      format: ExportFormat
    }) => exportSubmissions(formSchemaId, format),
    onSuccess: (result) => {
      downloadBlob(result.blob, result.filename)
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Webhooks
// ---------------------------------------------------------------------------

export function useCreateWebhook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      formSchemaId,
      ...input
    }: CreateWebhookInput & { formSchemaId: string }) =>
      createWebhook(formSchemaId, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: formulareKeys.webhooks.listBySchema(variables.formSchemaId),
      })
    },
  })
}

export function useUpdateWebhook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateWebhookInput & { id: string }) =>
      updateWebhook(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({
        queryKey: formulareKeys.webhooks.detail(variables.id),
      })
      qc.invalidateQueries({ queryKey: formulareKeys.webhooks.all })
    },
  })
}

export function useDeleteWebhook() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, formSchemaId: _formSchemaId }: { id: string; formSchemaId?: string }) =>
      deleteWebhook(id),
    onSuccess: (_data, variables) => {
      if (variables.formSchemaId) {
        qc.invalidateQueries({
          queryKey: formulareKeys.webhooks.listBySchema(variables.formSchemaId),
        })
      } else {
        qc.invalidateQueries({ queryKey: formulareKeys.webhooks.all })
      }
    },
  })
}
