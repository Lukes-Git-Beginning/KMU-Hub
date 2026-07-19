/**
 * RBAC R-3: finance:amounts:view helper.
 *
 * Two exports:
 * - useAmountsVisible()  — boolean hook, one subscription per component.
 * - maskedAmount()       — pure helper; replaces formatted string with '•••'
 *                          when the capability is absent.
 *
 * Usage:
 *   const amountsVisible = useAmountsVisible()
 *   <span title={!amountsVisible ? t('rbac.gate.amountsHidden') : undefined}>
 *     {maskedAmount(amountsVisible, formatMoney(total, currency))}
 *   </span>
 */
import { useHasCapability } from '@/hooks/useCapability'

export function useAmountsVisible(): boolean {
  return useHasCapability('finance:amounts:view')
}

/**
 * Returns `formatted` when `visible` is true, otherwise the mask string '•••'.
 * The literal '•••' is intentionally not a translated string — it is a visual
 * mask, not UI copy. `t('rbac.gate.amountsHidden')` is used as a tooltip.
 */
export function maskedAmount(visible: boolean, formatted: string): string {
  return visible ? formatted : '•••'
}
