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
]
