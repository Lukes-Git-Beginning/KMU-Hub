/**
 * VsChip — der Chip, mit dem eine Werteliste-Option (Status, Priorität, Stufe …)
 * in einer Liste oder Detail-Ansicht erscheint.
 *
 * Trägt die Option eine Farbe aus dem Wertelisten-Editor, tönt der Chip sich mit
 * ihr ein — Umfärben im Editor ist damit sofort im Modul sichtbar. Ohne Farbe
 * fällt er auf die semantische Tailwind-Klasse des Moduls zurück. So sieht der
 * Chip gleich aus, ob ein Tenant die Farben angefasst hat oder nicht.
 *
 * Liegt bewusst in `shared/`: jedes Modul, das Wertelisten rendert, braucht ihn,
 * und Chip-Form/Fallback sollen an genau einer Stelle entschieden werden statt
 * pro Modul zu driften (Rollout-Vorbereitung, Paket C1).
 *
 * ```tsx
 * const { labels, colorOf } = useModuleValueSet('helpdesk', 'ticket_status')
 * <VsChip label={labels[t.status]} color={colorOf(t.status)} fallbackClass={statusColors[t.status]} />
 * ```
 */
export function VsChip({
  label,
  color,
  fallbackClass,
  className = '',
}: {
  /** Angezeigter Text — bereits übersetzt bzw. aus der Werteliste aufgelöst. */
  label: string
  /** Farbe der Werteliste-Option (CSS-Farbe). Fehlt sie, greift `fallbackClass`. */
  color?: string
  /** Tailwind-Klassen für den Fall ohne eigene Farbe (z. B. `bg-info-light text-info`). */
  fallbackClass?: string
  /** Zusätzliche Klassen der Aufrufstelle, etwa eine feste Breite in Tabellen. */
  className?: string
}) {
  const base = 'inline-block rounded-full px-2 py-0.5 text-[10px] font-medium'
  if (color) {
    return (
      <span
        className={`${base} ${className}`}
        style={{ backgroundColor: `color-mix(in srgb, ${color} 16%, transparent)`, color }}
      >
        {label}
      </span>
    )
  }
  return <span className={`${base} ${fallbackClass ?? ''} ${className}`}>{label}</span>
}
