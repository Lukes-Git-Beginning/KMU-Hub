import { useTranslation } from 'react-i18next'
import type { VisualizationType } from '@/api/berichte-types'
import { VIZ_GALLERY } from './builder-utils'

interface VizSwitcherProps {
  value: VisualizationType
  /** True when the value was auto-suggested (shows a subtle hint). */
  auto?: boolean
  onSelect: (viz: VisualizationType) => void
}

export function VizSwitcher({ value, auto, onSelect }: VizSwitcherProps) {
  const { t } = useTranslation()
  return (
    <div>
      <div className="mb-1.5 flex items-center justify-between">
        <label className="text-xs font-medium text-muted-foreground">
          {t('berichte.builder.viz.label', { defaultValue: 'Visualisierung' })}
        </label>
        {auto && (
          <span className="text-[10px] text-muted-foreground/70">
            {t('berichte.builder.viz.auto', { defaultValue: 'automatisch' })}
          </span>
        )}
      </div>
      <div className="grid grid-cols-4 gap-1.5">
        {VIZ_GALLERY.map((v) => {
          const Icon = v.icon
          const active = value === v.type
          return (
            <button
              key={v.type}
              type="button"
              onClick={() => onSelect(v.type)}
              title={`${t(v.labelKey, { defaultValue: v.label })} — ${v.hint}`}
              className={`flex flex-col items-center gap-1 rounded-lg border p-2 transition-colors ${
                active
                  ? 'border-primary bg-primary-light text-primary'
                  : 'border-border text-muted-foreground hover:border-primary/30 hover:bg-secondary'
              }`}
            >
              <Icon className="h-4 w-4" />
              <span className="text-[10px] leading-none">{t(v.labelKey, { defaultValue: v.label })}</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
