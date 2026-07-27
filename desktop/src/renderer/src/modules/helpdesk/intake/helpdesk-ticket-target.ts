/**
 * Helpdesk intake target — the FIRST consumer of the shared intake engine.
 *
 * Declares how an editor-built form maps onto a Helpdesk ticket: the field roles
 * a form author can assign (subject/description/priority/category/requester) and
 * the `build`/`create` that turn a submission into a `CreateTicketInput` on the
 * wire. Every unmarked field flows into `custom_fields` (handled by the engine).
 *
 * Registered from `register-intake.ts` (imported once at app start) so the form
 * builder in the Formulare module can offer "→ Helpdesk-Ticket" without either
 * module importing the other.
 */
import { createTicket } from '@/api/helpdesk-client'
import type { CreateTicketInput, TicketPriority } from '@/api/helpdesk-types'
import type { IntakeBuildContext, IntakeTargetDef } from '@/components/shared/intake'

export const HELPDESK_TICKET_TARGET_ID = 'helpdesk_ticket'

/** Field-role keys understood by the Helpdesk target. */
export const HELPDESK_TICKET_ROLES = {
  subject: 'subject',
  description: 'description',
  priority: 'priority',
  category: 'category',
  requesterName: 'requester_name',
  requesterEmail: 'requester_email',
} as const

/** Coerce a free answer (German label or wire value) into a TicketPriority. */
function coercePriority(value: unknown): TicketPriority | undefined {
  if (value == null || value === '') return undefined
  const s = String(value).trim().toLowerCase()
  const map: Record<string, TicketPriority> = {
    low: 'low',
    niedrig: 'low',
    normal: 'normal',
    mittel: 'normal',
    medium: 'normal',
    high: 'high',
    hoch: 'high',
    urgent: 'urgent',
    dringend: 'urgent',
    kritisch: 'urgent',
  }
  return map[s]
}

function str(value: unknown): string | undefined {
  if (value == null) return undefined
  const s = String(value).trim()
  return s === '' ? undefined : s
}

export const helpdeskTicketTarget: IntakeTargetDef<CreateTicketInput> = {
  id: HELPDESK_TICKET_TARGET_ID,
  labelKey: 'intake.target.helpdeskTicket',
  roles: [
    { role: HELPDESK_TICKET_ROLES.subject, labelKey: 'intake.role.subject' },
    { role: HELPDESK_TICKET_ROLES.description, labelKey: 'intake.role.description' },
    {
      role: HELPDESK_TICKET_ROLES.priority,
      labelKey: 'intake.role.priority',
      fieldTypes: ['select', 'radio'],
    },
    {
      role: HELPDESK_TICKET_ROLES.category,
      labelKey: 'intake.role.category',
      fieldTypes: ['select', 'radio', 'text'],
    },
    {
      role: HELPDESK_TICKET_ROLES.requesterName,
      labelKey: 'intake.role.requesterName',
      fieldTypes: ['text'],
    },
    {
      role: HELPDESK_TICKET_ROLES.requesterEmail,
      labelKey: 'intake.role.requesterEmail',
      fieldTypes: ['email', 'text'],
    },
  ],

  build: (mapped, extras, context: IntakeBuildContext): CreateTicketInput => {
    const subject = str(mapped[HELPDESK_TICKET_ROLES.subject]) ?? 'Anfrage über Formular'
    const requesterName =
      str(mapped[HELPDESK_TICKET_ROLES.requesterName]) ?? context.requester?.name
    const requesterEmail =
      str(mapped[HELPDESK_TICKET_ROLES.requesterEmail]) ?? context.requester?.email

    // Origin: internal self-service submissions belong to the logged-in user;
    // everything else (public link, builder preview) counts as external intake.
    const channel = context.channel ?? 'external'
    const isExternal =
      context.requester?.isExternal ?? channel !== 'selfservice'

    return {
      subject,
      description: str(mapped[HELPDESK_TICKET_ROLES.description]),
      category: str(mapped[HELPDESK_TICKET_ROLES.category]),
      priority: coercePriority(mapped[HELPDESK_TICKET_ROLES.priority]),
      channel: channel as CreateTicketInput['channel'],
      requester_id: context.requester?.id,
      requester_name: requesterName,
      requester_email: requesterEmail,
      requester_is_external: isExternal,
      custom_fields: Object.keys(extras).length > 0 ? extras : undefined,
    }
  },

  create: (record) => createTicket(record),

  invalidateKeys: [['helpdesk', 'tickets']],
}
