/**
 * Reusable sort control: pick a field AND direction (asc/desc) from a dropdown.
 *
 * Designed to be dropped into any module list (contacts, deals, leads,
 * activities, …) so sorting behaves consistently everywhere. Keep the field
 * keys stable — callers map them to their own comparators / API sort params.
 */
import { useTranslation } from 'react-i18next'
import { ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib'

export type SortDirection = 'asc' | 'desc'

export interface SortFieldOption {
  value: string
  label: string
}

interface SortMenuProps {
  options: SortFieldOption[]
  field: string
  direction: SortDirection
  onChange: (field: string, direction: SortDirection) => void
  align?: 'start' | 'end'
  triggerClassName?: string
}

export function SortMenu({
  options,
  field,
  direction,
  onChange,
  align = 'end',
  triggerClassName,
}: SortMenuProps) {
  const { t } = useTranslation()
  const current = options.find((o) => o.value === field)
  const DirIcon = direction === 'asc' ? ArrowUp : ArrowDown

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          className={cn(
            'flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors',
            triggerClassName
          )}
          title={t('common.sort.sortBy')}
          aria-label={t('common.sort.sortBy')}
        >
          <ArrowUpDown className="h-4 w-4" />
          {current && <span className="hidden md:inline">{current.label}</span>}
          {current && <DirIcon className="h-3.5 w-3.5" />}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="min-w-[180px]">
        <DropdownMenuLabel>{t('common.sort.sortBy')}</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={field} onValueChange={(v) => onChange(v, direction)}>
          {options.map((o) => (
            <DropdownMenuRadioItem key={o.value} value={o.value}>
              {o.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>{t('common.sort.direction')}</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={direction} onValueChange={(v) => onChange(field, v as SortDirection)}>
          <DropdownMenuRadioItem value="asc">
            <ArrowUp className="mr-2 h-3.5 w-3.5" />
            {t('common.sort.ascending')}
          </DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="desc">
            <ArrowDown className="mr-2 h-3.5 w-3.5" />
            {t('common.sort.descending')}
          </DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
