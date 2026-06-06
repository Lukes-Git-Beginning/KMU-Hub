/**
 * Shared visual tokens for the leads inbox (temperature ampel colors).
 */
import type { LeadTemperature } from '@/api/hooks/useLeads'

export const TEMP_COLORS: Record<LeadTemperature, string> = {
  hot: '#ef4444',
  warm: '#f59e0b',
  cold: '#3b82f6',
}
