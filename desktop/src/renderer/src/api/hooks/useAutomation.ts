/**
 * TanStack Query hooks for the Automation module.
 *
 * Query keys follow the pattern ['automations', domain, ...params] for consistent
 * cache invalidation. Mutations use optimistic updates for enable/disable and
 * invalidation for CRUD operations.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import * as automationClient from '../automation-client'
import type {
  AutomationListParams,
  AutomationListResponse,
  ExecutionListParams,
  ConditionConfig,
  TemplateCategory,
} from '../automation-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const automationKeys = {
  all: ['automations'] as const,
  list: (params?: AutomationListParams) =>
    ['automations', 'list', params] as const,
  detail: (id: string) => ['automations', 'detail', id] as const,
  executions: (automationId: string, params?: ExecutionListParams) =>
    ['automations', 'executions', automationId, params] as const,
  execution: (executionId: string) =>
    ['automations', 'execution', executionId] as const,
  triggers: () => ['automations', 'triggers'] as const,
  actions: () => ['automations', 'actions'] as const,
  templates: (category?: string) =>
    ['automations', 'templates', category] as const,
  stats: () => ['automations', 'stats'] as const,
}

// ---------------------------------------------------------------------------
// Query hooks
// ---------------------------------------------------------------------------

export function useAutomations(params?: AutomationListParams) {
  return useQuery({
    queryKey: automationKeys.list(params),
    queryFn: () => automationClient.listAutomations(params),
    staleTime: 30 * 1000, // 30s
  })
}

export function useAutomation(id: string) {
  return useQuery({
    queryKey: automationKeys.detail(id),
    queryFn: () => automationClient.getAutomation(id),
    enabled: !!id,
  })
}

export function useAutomationExecutions(
  automationId: string,
  params?: ExecutionListParams,
) {
  return useQuery({
    queryKey: automationKeys.executions(automationId, params),
    queryFn: () => automationClient.listExecutions(automationId, params),
    enabled: !!automationId,
    staleTime: 30 * 1000,
  })
}

export function useAutomationExecution(executionId: string) {
  return useQuery({
    queryKey: automationKeys.execution(executionId),
    queryFn: () => automationClient.getExecution(executionId),
    enabled: !!executionId,
  })
}

export function useTriggerDefinitions() {
  return useQuery({
    queryKey: automationKeys.triggers(),
    queryFn: () => automationClient.listTriggerDefinitions(),
    staleTime: 5 * 60 * 1000, // 5min -- rarely changes
  })
}

export function useActionDefinitions() {
  return useQuery({
    queryKey: automationKeys.actions(),
    queryFn: () => automationClient.listActionDefinitions(),
    staleTime: 5 * 60 * 1000, // 5min
  })
}

export function useAutomationTemplates(category?: TemplateCategory) {
  return useQuery({
    queryKey: automationKeys.templates(category),
    queryFn: () => automationClient.listTemplates(category),
    staleTime: 5 * 60 * 1000, // 5min
  })
}

export function useAutomationStats() {
  return useQuery({
    queryKey: automationKeys.stats(),
    queryFn: () => automationClient.getAutomationStats(),
    staleTime: 60 * 1000, // 60s
  })
}

// ---------------------------------------------------------------------------
// Mutation hooks
// ---------------------------------------------------------------------------

export function useCreateAutomation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Parameters<typeof automationClient.createAutomation>[0]) =>
      automationClient.createAutomation(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: automationKeys.all })
      toast.success('Automatisierung erstellt')
    },
    onError: () => {
      toast.error('Fehler beim Erstellen der Automatisierung')
    },
  })
}

export function useUpdateAutomation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      ...data
    }: { id: string } & Parameters<typeof automationClient.updateAutomation>[1]) =>
      automationClient.updateAutomation(id, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: automationKeys.list() })
      qc.invalidateQueries({ queryKey: automationKeys.detail(vars.id) })
      toast.success('Automatisierung aktualisiert')
    },
    onError: () => {
      toast.error('Fehler beim Aktualisieren der Automatisierung')
    },
  })
}

export function useDeleteAutomation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => automationClient.deleteAutomation(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: automationKeys.all })
      toast.success('Automatisierung geloescht')
    },
    onError: () => {
      toast.error('Fehler beim Loeschen der Automatisierung')
    },
  })
}

export function useEnableAutomation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => automationClient.enableAutomation(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: automationKeys.all })
      // Optimistic update: set is_active=true in list caches
      qc.setQueriesData<AutomationListResponse>(
        { queryKey: ['automations', 'list'] },
        (old) => {
          if (!old) return old
          return {
            ...old,
            automations: old.automations.map((a) =>
              a.id === id ? { ...a, is_active: true } : a,
            ),
          }
        },
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: automationKeys.all })
    },
    onError: () => {
      qc.invalidateQueries({ queryKey: automationKeys.all })
      toast.error('Fehler beim Aktivieren der Automatisierung')
    },
  })
}

export function useDisableAutomation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => automationClient.disableAutomation(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: automationKeys.all })
      // Optimistic update: set is_active=false in list caches
      qc.setQueriesData<AutomationListResponse>(
        { queryKey: ['automations', 'list'] },
        (old) => {
          if (!old) return old
          return {
            ...old,
            automations: old.automations.map((a) =>
              a.id === id ? { ...a, is_active: false } : a,
            ),
          }
        },
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: automationKeys.all })
    },
    onError: () => {
      qc.invalidateQueries({ queryKey: automationKeys.all })
      toast.error('Fehler beim Deaktivieren der Automatisierung')
    },
  })
}

export function useCreateFromTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      templateId,
      name,
    }: {
      templateId: string
      name: string
    }) => automationClient.createFromTemplate(templateId, { name }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: automationKeys.all })
      toast.success('Automatisierung aus Vorlage erstellt')
    },
    onError: () => {
      toast.error('Fehler beim Erstellen aus Vorlage')
    },
  })
}

export function useTestCondition() {
  return useMutation({
    mutationFn: (data: {
      condition: ConditionConfig
      sample_env: Record<string, unknown>
    }) => automationClient.testCondition(data),
  })
}

export function useDryRunAutomation() {
  return useMutation({
    mutationFn: (data: {
      automation_id: string
      sample_env: Record<string, unknown>
    }) => automationClient.dryRunAutomation(data),
  })
}
