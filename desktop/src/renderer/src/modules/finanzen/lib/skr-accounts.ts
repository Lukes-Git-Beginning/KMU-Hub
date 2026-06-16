/**
 * SKR03 / SKR04 chart of accounts — leichtes Sachkonto-Mapping (finanzen P2).
 *
 * Strategie (siehe finanzen-buchhaltung-strategy.md): Cosmi ersetzt KEINE
 * Buchhaltung. Wir hängen jeder Ausgabe nur ein Sachkonto an, damit der spätere
 * DATEV-EXTF-Export (P3) die nötigen Daten hat. Keine Buchungssätze, kein
 * Soll/Haben. Der Kontenrahmen (SKR03 für DE-Bilanzierer ist DATEV-Default,
 * SKR04 = Abschlussgliederung) ist eine tenant-weite Einstellung.
 *
 * Die Kontonummern/-bezeichnungen sind die deutschen Standard-DATEV-Konten und
 * bleiben sprachunabhängig (Eigennamen des Kontenrahmens). Übersetzt werden nur
 * die Ausgaben-Kategorien.
 */

export type ChartFramework = 'SKR03' | 'SKR04'

export interface SkrAccount {
  /** Sachkonto-Nummer, z. B. "4930". */
  number: string
  /** Deutsche Kontobezeichnung, z. B. "Bürobedarf". */
  label: string
}

/**
 * Kanonische Ausgaben-Kategorien. `value` ist gleichzeitig der i18n-Suffix
 * (`buchhaltung.categories.<value>`) und der in `Expense.category` gespeicherte
 * Wert. Pro Kategorie ein Default-Sachkonto je Kontenrahmen.
 */
export interface ExpenseCategory {
  value: string
  skr03: string
  skr04: string
}

export const EXPENSE_CATEGORIES: ExpenseCategory[] = [
  { value: 'Büromaterial', skr03: '4930', skr04: '6815' },
  { value: 'Software', skr03: '4806', skr04: '6495' },
  { value: 'IT-Infrastruktur', skr03: '4900', skr04: '6850' },
  { value: 'Hardware', skr03: '0480', skr04: '0670' },
  { value: 'Bewirtung', skr03: '4650', skr04: '6640' },
  { value: 'Reisekosten', skr03: '4660', skr04: '6650' },
  { value: 'Marketing', skr03: '4600', skr04: '6600' },
  { value: 'Weiterbildung', skr03: '4945', skr04: '6821' },
  { value: 'Beratung', skr03: '4950', skr04: '6825' },
  { value: 'Versicherung', skr03: '4360', skr04: '6400' },
  { value: 'Bankgebühren', skr03: '4970', skr04: '6855' },
  { value: 'Sonstiges', skr03: '4900', skr04: '6850' },
]

/**
 * Häufige Aufwands-Sachkonten je Kontenrahmen — die Auswahl für das
 * Kontierungs-Dropdown. Bewusst kuratiert (kein vollständiger Kontenrahmen).
 */
export const SKR_ACCOUNTS: Record<ChartFramework, SkrAccount[]> = {
  SKR03: [
    { number: '4360', label: 'Versicherungen' },
    { number: '4380', label: 'Beiträge' },
    { number: '4600', label: 'Werbekosten' },
    { number: '4650', label: 'Bewirtungskosten' },
    { number: '4660', label: 'Reisekosten Arbeitnehmer' },
    { number: '4670', label: 'Reisekosten Unternehmer' },
    { number: '4806', label: 'Wartungskosten Hard- und Software' },
    { number: '4900', label: 'Sonstige betriebliche Aufwendungen' },
    { number: '4910', label: 'Porto' },
    { number: '4920', label: 'Telefon' },
    { number: '4930', label: 'Bürobedarf' },
    { number: '4940', label: 'Zeitschriften, Bücher' },
    { number: '4945', label: 'Fortbildungskosten' },
    { number: '4950', label: 'Rechts- und Beratungskosten' },
    { number: '4970', label: 'Nebenkosten des Geldverkehrs' },
    { number: '0480', label: 'Geringwertige Wirtschaftsgüter' },
  ],
  SKR04: [
    { number: '6400', label: 'Versicherungen' },
    { number: '6420', label: 'Beiträge' },
    { number: '6600', label: 'Werbekosten' },
    { number: '6640', label: 'Bewirtungskosten' },
    { number: '6650', label: 'Reisekosten Arbeitnehmer' },
    { number: '6670', label: 'Reisekosten Unternehmer' },
    { number: '6495', label: 'Wartungskosten Hard- und Software' },
    { number: '6800', label: 'Porto' },
    { number: '6805', label: 'Telefon' },
    { number: '6815', label: 'Bürobedarf' },
    { number: '6820', label: 'Zeitschriften, Bücher' },
    { number: '6821', label: 'Fortbildungskosten' },
    { number: '6825', label: 'Rechts- und Beratungskosten' },
    { number: '6850', label: 'Sonstige betriebliche Aufwendungen' },
    { number: '6855', label: 'Nebenkosten des Geldverkehrs' },
    { number: '0670', label: 'Geringwertige Wirtschaftsgüter' },
  ],
}

/** Schlägt das Default-Sachkonto für eine Kategorie im gewählten Kontenrahmen vor. */
export function suggestAccount(category: string, framework: ChartFramework): string | undefined {
  const cat = EXPENSE_CATEGORIES.find((c) => c.value === category)
  if (!cat) return undefined
  return framework === 'SKR03' ? cat.skr03 : cat.skr04
}

/** Liefert "4930 · Bürobedarf" für die Anzeige; fällt auf die nackte Nummer zurück. */
export function formatAccount(number: string | undefined, framework: ChartFramework): string {
  if (!number) return ''
  const acc = SKR_ACCOUNTS[framework].find((a) => a.number === number)
  return acc ? `${acc.number} · ${acc.label}` : number
}
