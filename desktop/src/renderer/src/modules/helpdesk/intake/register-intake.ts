/**
 * Registers the Helpdesk intake target into the shared engine.
 *
 * Imported once at app start (main.tsx) as a side effect so the Formulare form
 * builder can offer "→ Helpdesk-Ticket" and any channel can dispatch to it —
 * without Formulare and Helpdesk importing each other.
 */
import { registerIntakeTarget } from '@/components/shared/intake'
import { helpdeskTicketTarget } from './helpdesk-ticket-target'

registerIntakeTarget(helpdeskTicketTarget)
