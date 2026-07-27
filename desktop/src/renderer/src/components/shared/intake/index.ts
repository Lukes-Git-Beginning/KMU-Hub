/**
 * Shared intake engine — the editor-built-form → module-record substrate.
 *
 *   import {
 *     registerIntakeTarget, getIntakeTarget, listIntakeTargets, getIntakeRoles,
 *     mapSubmissionToRecord, useIntakeSubmit, INTAKE_ROLE_EXTRA,
 *   } from '@/components/shared/intake'
 *
 * Helpdesk (form → ticket) is the first consumer; other request-shaped modules
 * (HR requests, procurement/maintenance requests, complaints, …) register their
 * own target the same way. Never import a module from here.
 */
export * from './types'
export { registerIntakeTarget, getIntakeTarget, listIntakeTargets, getIntakeRoles } from './registry'
export { mapSubmissionToRecord, slugifyKey } from './engine'
export { useIntakeSubmit } from './useIntakeSubmit'
export type { IntakeDispatchArgs, IntakeDispatchResult } from './useIntakeSubmit'
export { IntakeFormFill } from './IntakeFormFill'
export type { IntakeFormFillProps } from './IntakeFormFill'
