/**
 * TanStack Query hooks for the Plugin system module.
 *
 * Query keys follow the pattern ['plugins', domain, ...params] for
 * consistent cache invalidation. Mutations invalidate related queries
 * and show German toast notifications.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import i18next from 'i18next'
import * as pluginClient from '../plugin-client'
import type {
  CreateManifestRequest,
  InstallPluginRequest,
  ApprovePermissionsRequest,
  UpdateSettingsRequest,
  CreateValidationRuleRequest,
  UpdateValidationRuleRequest,
  CreateWorkflowRuleRequest,
  UpdateWorkflowRuleRequest,
  ApplyTemplateRequest,
} from '../plugin-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const pluginKeys = {
  all: ['plugins'] as const,
  manifests: () => ['plugins', 'manifests'] as const,
  manifest: (id: string) => ['plugins', 'manifests', id] as const,
  installations: () => ['plugins', 'installations'] as const,
  permissions: (installationId: string) =>
    ['plugins', 'permissions', installationId] as const,
  settings: (installationId: string) =>
    ['plugins', 'settings', installationId] as const,
  settingsSchema: (installationId: string) =>
    ['plugins', 'settings-schema', installationId] as const,
  validationRules: (installationId?: string) =>
    ['plugins', 'validation-rules', installationId] as const,
  workflowRules: (installationId?: string) =>
    ['plugins', 'workflow-rules', installationId] as const,
  templates: () => ['plugins', 'templates'] as const,
  executionLogs: (installationId?: string, limit?: number) =>
    ['plugins', 'execution-logs', installationId, limit] as const,
}

// ---------------------------------------------------------------------------
// Manifest queries
// ---------------------------------------------------------------------------

export function usePluginManifests() {
  return useQuery({
    queryKey: pluginKeys.manifests(),
    queryFn: () => pluginClient.listManifests(),
    select: (data) => data.manifests,
    staleTime: 30 * 1000,
  })
}

export function usePluginManifest(id: string) {
  return useQuery({
    queryKey: pluginKeys.manifest(id),
    queryFn: () => pluginClient.getManifest(id),
    staleTime: 30 * 1000,
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Manifest mutations
// ---------------------------------------------------------------------------

export function useCreateManifest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateManifestRequest) =>
      pluginClient.createManifest(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pluginKeys.manifests() })
      toast.success(i18next.t('api.plugins.manifestCreated'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.manifestCreate'))
    },
  })
}

export function useDeleteManifest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => pluginClient.deleteManifest(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pluginKeys.manifests() })
      toast.success(i18next.t('api.plugins.manifestDeleted'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.manifestDelete'))
    },
  })
}

// ---------------------------------------------------------------------------
// Installation queries
// ---------------------------------------------------------------------------

export function usePluginInstallations() {
  return useQuery({
    queryKey: pluginKeys.installations(),
    queryFn: () => pluginClient.listInstallations(),
    select: (data) => data.installations,
    staleTime: 15 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Installation mutations
// ---------------------------------------------------------------------------

export function useInstallPlugin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: InstallPluginRequest) =>
      pluginClient.installPlugin(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pluginKeys.installations() })
      toast.success(i18next.t('api.plugins.installed'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.install'))
    },
  })
}

export function useEnablePlugin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (installationId: string) =>
      pluginClient.enablePlugin(installationId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pluginKeys.installations() })
      toast.success(i18next.t('api.plugins.enabled'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.enable'))
    },
  })
}

export function useDisablePlugin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (installationId: string) =>
      pluginClient.disablePlugin(installationId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pluginKeys.installations() })
      toast.success(i18next.t('api.plugins.disabled'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.disable'))
    },
  })
}

export function useUninstallPlugin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (installationId: string) =>
      pluginClient.uninstallPlugin(installationId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pluginKeys.installations() })
      toast.success(i18next.t('api.plugins.uninstalled'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.uninstall'))
    },
  })
}

// ---------------------------------------------------------------------------
// Permission queries & mutations
// ---------------------------------------------------------------------------

export function usePluginPermissions(installationId: string) {
  return useQuery({
    queryKey: pluginKeys.permissions(installationId),
    queryFn: () => pluginClient.listPermissions(installationId),
    staleTime: 30 * 1000,
    enabled: !!installationId,
  })
}

export function useApprovePermissions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      installationId,
      data,
    }: {
      installationId: string
      data: ApprovePermissionsRequest
    }) => pluginClient.approvePermissions(installationId, data),
    onSuccess: (_, variables) => {
      qc.invalidateQueries({
        queryKey: pluginKeys.permissions(variables.installationId),
      })
      qc.invalidateQueries({ queryKey: pluginKeys.installations() })
      toast.success(i18next.t('api.plugins.permissionsApproved'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.permissionsApprove'))
    },
  })
}

// ---------------------------------------------------------------------------
// Settings queries & mutations
// ---------------------------------------------------------------------------

export function usePluginSettings(installationId: string) {
  return useQuery({
    queryKey: pluginKeys.settings(installationId),
    queryFn: () => pluginClient.getSettings(installationId),
    staleTime: 30 * 1000,
    enabled: !!installationId,
  })
}

export function usePluginSettingsSchema(installationId: string) {
  return useQuery({
    queryKey: pluginKeys.settingsSchema(installationId),
    queryFn: () => pluginClient.getSettingsSchema(installationId),
    select: (data) => data.schema,
    staleTime: 60 * 1000,
    enabled: !!installationId,
  })
}

export function useUpdatePluginSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      installationId,
      data,
    }: {
      installationId: string
      data: UpdateSettingsRequest
    }) => pluginClient.updateSettings(installationId, data),
    onSuccess: (_, variables) => {
      qc.invalidateQueries({
        queryKey: pluginKeys.settings(variables.installationId),
      })
      toast.success(i18next.t('api.plugins.settingsSaved'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.settingsSave'))
    },
  })
}

// ---------------------------------------------------------------------------
// Validation rule queries & mutations
// ---------------------------------------------------------------------------

export function useValidationRules(installationId?: string) {
  return useQuery({
    queryKey: pluginKeys.validationRules(installationId),
    queryFn: () => pluginClient.listValidationRules(installationId),
    select: (data) => data.rules,
    staleTime: 15 * 1000,
  })
}

export function useCreateValidationRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateValidationRuleRequest) =>
      pluginClient.createValidationRule(data),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['plugins', 'validation-rules'],
      })
      toast.success(i18next.t('api.plugins.validationRuleCreated'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.validationRuleCreate'))
    },
  })
}

export function useUpdateValidationRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: UpdateValidationRuleRequest
    }) => pluginClient.updateValidationRule(id, data),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['plugins', 'validation-rules'],
      })
      toast.success(i18next.t('api.plugins.validationRuleUpdated'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.validationRuleUpdate'))
    },
  })
}

export function useDeleteValidationRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => pluginClient.deleteValidationRule(id),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['plugins', 'validation-rules'],
      })
      toast.success(i18next.t('api.plugins.validationRuleDeleted'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.validationRuleDelete'))
    },
  })
}

// ---------------------------------------------------------------------------
// Workflow rule queries & mutations
// ---------------------------------------------------------------------------

export function useWorkflowRules(installationId?: string) {
  return useQuery({
    queryKey: pluginKeys.workflowRules(installationId),
    queryFn: () => pluginClient.listWorkflowRules(installationId),
    select: (data) => data.rules,
    staleTime: 15 * 1000,
  })
}

export function useCreateWorkflowRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateWorkflowRuleRequest) =>
      pluginClient.createWorkflowRule(data),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['plugins', 'workflow-rules'],
      })
      toast.success(i18next.t('api.plugins.workflowRuleCreated'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.workflowRuleCreate'))
    },
  })
}

export function useUpdateWorkflowRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: string
      data: UpdateWorkflowRuleRequest
    }) => pluginClient.updateWorkflowRule(id, data),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['plugins', 'workflow-rules'],
      })
      toast.success(i18next.t('api.plugins.workflowRuleUpdated'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.workflowRuleUpdate'))
    },
  })
}

export function useDeleteWorkflowRule() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => pluginClient.deleteWorkflowRule(id),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: ['plugins', 'workflow-rules'],
      })
      toast.success(i18next.t('api.plugins.workflowRuleDeleted'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.workflowRuleDelete'))
    },
  })
}

// ---------------------------------------------------------------------------
// Industry template queries & mutations
// ---------------------------------------------------------------------------

export function useIndustryTemplates() {
  return useQuery({
    queryKey: pluginKeys.templates(),
    queryFn: () => pluginClient.listTemplates(),
    select: (data) => data.templates,
    staleTime: 60 * 1000,
  })
}

export function useApplyTemplate() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: ApplyTemplateRequest) =>
      pluginClient.applyTemplate(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pluginKeys.installations() })
      qc.invalidateQueries({ queryKey: ['plugins', 'validation-rules'] })
      qc.invalidateQueries({ queryKey: ['plugins', 'workflow-rules'] })
      toast.success(i18next.t('api.plugins.templateApplied'))
    },
    onError: () => {
      toast.error(i18next.t('api.plugins.error.templateApply'))
    },
  })
}

// ---------------------------------------------------------------------------
// Execution log queries
// ---------------------------------------------------------------------------

export function useExecutionLogs(installationId?: string, limit = 50) {
  return useQuery({
    queryKey: pluginKeys.executionLogs(installationId, limit),
    queryFn: () => pluginClient.listExecutionLogs(installationId, limit),
    select: (data) => data.logs,
    staleTime: 10 * 1000,
  })
}
