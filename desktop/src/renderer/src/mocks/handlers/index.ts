/**
 * Aggregates all MSW handler arrays into a single export.
 * demo-mode.ts imports this to register all handlers with the worker.
 */
import { authHandlers } from './auth'
import { crmHandlers } from './crm'
import { workHandlers } from './work'
import { chatHandlers } from './chat'
import { emailHandlers } from './email'
import { calendarHandlers } from './calendar'
import { financeHandlers } from './finance'
import { documentHandlers } from './documents'
import { automationHandlers } from './automation'
import { securityHandlers } from './security'
import { teamHandlers } from './team'
import { notificationHandlers } from './notifications'
import { inboxHandlers } from './inbox'
import { videoHandlers } from './video'
import { dashboardHandlers } from './dashboard'
import { settingsHandlers } from './settings'
import { hrHandlers } from './hr'
import { dialerHandlers } from './dialer'
import { berichteHandlers } from './berichte'
import { bookingHandlers } from './booking'
import { vermietungHandlers } from './vermietung'
import { wikiHandlers } from './wiki'
import { schichtenHandlers } from './schichten'
import { helpdeskHandlers } from './helpdesk'
import { inventarHandlers } from './inventar'
import { produktionHandlers } from './produktion'

export const handlers = [
  ...authHandlers,
  ...crmHandlers,
  ...workHandlers,
  ...chatHandlers,
  ...emailHandlers,
  ...calendarHandlers,
  ...financeHandlers,
  ...documentHandlers,
  ...automationHandlers,
  ...securityHandlers,
  ...teamHandlers,
  ...notificationHandlers,
  ...inboxHandlers,
  ...videoHandlers,
  ...dashboardHandlers,
  ...settingsHandlers,
  ...hrHandlers,
  ...dialerHandlers,
  ...berichteHandlers,
  ...bookingHandlers,
  ...vermietungHandlers,
  ...wikiHandlers,
  ...schichtenHandlers,
  ...helpdeskHandlers,
  ...inventarHandlers,
  ...produktionHandlers,
]
