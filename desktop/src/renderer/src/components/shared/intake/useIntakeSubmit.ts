/**
 * Shared intake engine — the dispatch hook.
 *
 * `useIntakeSubmit` wires the pure mapping to React Query: it maps a submission
 * to the module record, calls the target's `create`, then invalidates the
 * target's list queries so the module view refreshes. Any channel (agent preview,
 * internal self-service, external public page) can reuse it.
 */
import { useQueryClient } from '@tanstack/react-query'
import { useCallback } from 'react'
import { mapSubmissionToRecord } from './engine'
import { getIntakeTarget } from './registry'
import type { IntakeBuildContext, IntakeField } from './types'

export interface IntakeDispatchArgs {
  targetId: string
  fields: IntakeField[]
  answers: Record<string, unknown>
  context?: IntakeBuildContext
}

export interface IntakeDispatchResult {
  /** The created record's id (e.g. the new ticket id). */
  id: string
  record: { id: string }
}

export function useIntakeSubmit(): (args: IntakeDispatchArgs) => Promise<IntakeDispatchResult> {
  const queryClient = useQueryClient()

  return useCallback(
    async ({ targetId, fields, answers, context }: IntakeDispatchArgs) => {
      const target = getIntakeTarget(targetId)
      if (!target) throw new Error(`intake: no target registered for "${targetId}"`)

      const { record } = mapSubmissionToRecord(targetId, fields, answers, context ?? {})
      const created = await target.create(record)

      for (const key of target.invalidateKeys ?? []) {
        await queryClient.invalidateQueries({ queryKey: key })
      }

      return { id: created.id, record: created }
    },
    [queryClient],
  )
}
