import type { LucideIcon } from 'lucide-react'
import { SearchResultItem } from './SearchResultItem'

export interface GroupedResult {
  label: string
  items: Array<{
    id: string
    icon: LucideIcon
    title: string
    subtitle: string
    route: string
  }>
}

interface SearchResultGroupProps {
  group: GroupedResult
  activeIndex: number
  startIndex: number
  onSelect: (route: string) => void
  onHover: (index: number) => void
}

export function SearchResultGroup({
  group,
  activeIndex,
  startIndex,
  onSelect,
  onHover,
}: SearchResultGroupProps) {
  return (
    <div className="px-2 py-1.5">
      <div className="px-2 pb-1 text-xs font-medium text-muted-foreground">
        {group.label}
      </div>
      {group.items.map((item, i) => (
        <SearchResultItem
          key={item.id}
          icon={item.icon}
          title={item.title}
          subtitle={item.subtitle}
          active={activeIndex === startIndex + i}
          onClick={() => onSelect(item.route)}
          onMouseEnter={() => onHover(startIndex + i)}
        />
      ))}
    </div>
  )
}
