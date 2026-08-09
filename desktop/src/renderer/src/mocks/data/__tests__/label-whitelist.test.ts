/**
 * LABEL_WHITELIST-Wächter.
 *
 * Die Falle, die diesen Test nötig gemacht hat (Sessions #34–#36): `applyDraftToTenant`
 * schreibt beim Übernehmen **nur** Labels, deren Key in `LABEL_WHITELIST` steht. Ein
 * `<EditableText dkey="…" />` ohne Whitelist-Eintrag lässt sich im Editor umbenennen,
 * die Vorschau zeigt den neuen Namen — und beim Deploy wird er **still verworfen**.
 * Kein Fehler, kein Hinweis, die Umbenennung ist einfach weg. Genau so ist die
 * Spalten-Runde aufgelaufen.
 *
 * Der Test schließt die Lücke von beiden Seiten:
 *   1. jeder statisch instrumentierte `dkey` im Quellcode muss in der Whitelist stehen,
 *   2. jeder im Editor umbenennbare Registry-Key (labelKeys / listColumns / areas /
 *      statWidgets) ebenso,
 *   3. dynamische `dkey={ausdruck}`-Stellen müssen unten registriert sein, damit
 *      niemand eine neue Quelle einführt, die der Scan nicht sieht.
 *
 * Beim Instrumentieren eines neuen Moduls ist die Reihenfolge also: `EditableText`
 * setzen → Key in `LABEL_WHITELIST` nachtragen → Test läuft grün.
 */
import { describe, it, expect } from 'vitest'
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'
import { LABEL_WHITELIST } from '../customization'
import { EDITOR_MODULES } from '@/modules/admin/anpassungen/editor/editorModules'

/**
 * `import.meta.url` trägt unter jsdom kein `file:`-Schema, deshalb über cwd. Vitest
 * startet aus `desktop/`; der zweite Zweig fängt einen Lauf aus dem Repo-Root ab.
 */
const SRC_ROOT = [
  resolve(process.cwd(), 'src/renderer/src'),
  resolve(process.cwd(), 'desktop/src/renderer/src'),
].find(existsSync) as string

/**
 * Dateien, in denen `dkey` vorkommt, ohne eine Instrumentierung zu sein — die
 * Definition von `EditableText` selbst nennt den Namen im Doc-Kommentar.
 */
const SCAN_EXCLUDE = ['components/customization/EditorSurface.tsx']

/**
 * Registrierte dynamische Quellen: `<EditableText dkey={ausdruck} />`. Der Scan kann
 * den Wert nicht statisch auflösen, deshalb steht hier, woher er kommt und wer ihn
 * abdeckt. Neue dynamische Stelle → Eintrag hinzufügen und sicherstellen, dass ihre
 * Keys über die Registry (Punkt 2) oder direkt in der Whitelist landen.
 */
const KNOWN_DYNAMIC_SOURCES: Record<string, string> = {
  'item.dkey': 'helpdesk tab list — keys are the EditorModuleDef.areas labelKeys',
  'cat.labelKey': 'kontakte category sidebar — keys are kontakte.category.*',
  'item.labelKey': 'kontakte section bar — keys are the EditorModuleDef.areas labelKeys',
}

// ── Quellcode-Scan ───────────────────────────────────────────────────────────

function collectTsxFiles(dir: string, out: string[] = []): string[] {
  for (const entry of readdirSync(dir)) {
    if (entry === 'node_modules' || entry === '__tests__') continue
    const full = join(dir, entry)
    if (statSync(full).isDirectory()) collectTsxFiles(full, out)
    else if (entry.endsWith('.tsx')) out.push(full)
  }
  return out
}

interface Hit {
  key: string
  file: string
}

function scanInstrumentation(): { statik: Hit[]; dynamisch: Hit[] } {
  const statik: Hit[] = []
  const dynamisch: Hit[] = []

  for (const file of collectTsxFiles(SRC_ROOT)) {
    const rel = relative(SRC_ROOT, file).replace(/\\/g, '/')
    if (SCAN_EXCLUDE.includes(rel)) continue

    const source = readFileSync(file, 'utf8')
    if (!source.includes('dkey')) continue

    for (const m of source.matchAll(/dkey=(?:"([^"]+)"|'([^']+)')/g)) {
      statik.push({ key: (m[1] ?? m[2]) as string, file: rel })
    }
    for (const m of source.matchAll(/dkey=\{([^}]+)\}/g)) {
      dynamisch.push({ key: m[1].trim(), file: rel })
    }
  }

  return { statik, dynamisch }
}

// ── Tests ────────────────────────────────────────────────────────────────────

describe('LABEL_WHITELIST', () => {
  const whitelist = new Set(LABEL_WHITELIST)

  it('deckt jeden statisch instrumentierten dkey ab', () => {
    const { statik } = scanInstrumentation()

    // Absicherung gegen einen kaputten Scan: findet er nichts, ist der Test
    // wertlos und würde stumm grün bleiben.
    expect(statik.length).toBeGreaterThan(0)

    const fehlend = statik
      .filter((h) => !whitelist.has(h.key))
      .map((h) => `${h.key}  (${h.file})`)
      .sort()

    expect(
      [...new Set(fehlend)],
      'Diese Keys sind als <EditableText> instrumentiert, stehen aber nicht in ' +
        'LABEL_WHITELIST — ihre Umbenennung wird beim Übernehmen still verworfen. ' +
        'Nachtragen in mocks/data/customization.ts.',
    ).toEqual([])
  })

  it('deckt jeden im Editor umbenennbaren Registry-Key ab', () => {
    const fehlend: string[] = []

    for (const mod of EDITOR_MODULES) {
      const keys = [
        ...mod.labelKeys,
        ...(mod.listColumns ?? []).map((c) => c.labelKey),
        ...mod.areas.map((a) => a.labelKey),
        ...(mod.statWidgets ?? []).map((w) => w.labelKey),
      ]
      for (const key of keys) {
        // Editor-eigene Beschriftungen (customization.*) gehören dem Editor, nicht
        // dem Modul — sie sind nicht durch den Tenant umbenennbar.
        if (key.startsWith('customization.')) continue
        if (!whitelist.has(key)) fehlend.push(`${key}  (${mod.key})`)
      }
    }

    expect(
      [...new Set(fehlend)].sort(),
      'Diese Registry-Keys sind im Editor umbenennbar, überleben den Deploy aber ' +
        'nicht — in LABEL_WHITELIST nachtragen.',
    ).toEqual([])
  })

  it('kennt jede dynamische dkey-Quelle', () => {
    const { dynamisch } = scanInstrumentation()

    const unbekannt = dynamisch
      .filter((h) => !(h.key in KNOWN_DYNAMIC_SOURCES))
      .map((h) => `${h.key}  (${h.file})`)
      .sort()

    expect(
      [...new Set(unbekannt)],
      'Neue <EditableText dkey={ausdruck} />-Stelle: der Scan kann ihre Keys nicht ' +
        'lesen. In KNOWN_DYNAMIC_SOURCES eintragen und prüfen, dass die Keys in ' +
        'LABEL_WHITELIST stehen.',
    ).toEqual([])
  })

  it('enthält keine Modul-Identität', () => {
    // Modulnamen sind unveränderlich (Darien 2026-07-22) — stünde ein rbac.module.*
    // oder layout.navItems.* in der Whitelist, ließe sich dasselbe Modul unter drei
    // Namen führen.
    const identitaet = LABEL_WHITELIST.filter(
      (k) => k.startsWith('rbac.module.') || k.startsWith('layout.navItems.'),
    )
    expect(identitaet).toEqual([])
  })

  it('führt jeden Key nur einmal', () => {
    const doppelt = LABEL_WHITELIST.filter((k, i) => LABEL_WHITELIST.indexOf(k) !== i)
    expect([...new Set(doppelt)]).toEqual([])
  })
})
