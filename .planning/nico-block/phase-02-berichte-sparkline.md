# Phase 02 (Pilot) — Berichte: Mini-Trend (Sparkline) in den KPI-Karten

> **Modul:** berichte · **Risiko:** mittel (klar eingegrenzt) · **Backend:** nicht nötig — rein visuell mit vorhandenen Daten/Mock.
> Zweite **Pilot-Phase**: ein kleines, sehr sichtbares Feature mit exaktem Muster im Repo.

## Ziel
Jede **KPI-Karte** im Berichte-Dashboard bekommt unten einen kleinen **Trend-Verlauf (Sparkline)** — eine schlanke Linie, die den Verlauf der letzten Punkte zeigt. Das macht die Kennzahlen lebendiger, ohne ein großes Chart.

## Ist-Stand (was schon da ist)
- Karte: `desktop/src/renderer/src/modules/berichte/components/KPICard.tsx` (~71 Zeilen, klar abgegrenzt).
- Dashboard, das die Karten rendert: `modules/berichte/components/DashboardGrid.tsx` (KPIs aus `useDashboardKPIs`).
- **Fertige Utilities** (genau dafür):
  - `modules/berichte/utils/chartTheme.ts` → `useChartTheme()` (liefert Farben aus den CSS-Variablen, z.B. `theme.primary`, `theme.grid`).
  - `modules/berichte/utils/chartMotion.ts` → `usePrefersReducedMotion()`.
- `recharts` ist bereits Dependency (in `DashboardGrid.tsx` importiert).

## Muster-Vorlage (1:1 übernehmen)
In `DashboardGrid.tsx` (~Z. 180–205) steht ein fertiges `LineChart` mit `useChartTheme` + `usePrefersReducedMotion` + `CartesianGrid stroke={theme.grid}`. **Kopiere dieses Muster** in verkleinerter Form (ohne Achsen/Grid, feste kleine Höhe) in die KPICard.

## Schritte
1. `git pull`. App läuft, öffne `/berichte` → Dashboard mit KPI-Karten.
2. `KPICard.tsx`: einen optionalen Prop `sparklineData?: { value: number }[]` ergänzen.
3. Wenn `sparklineData` vorhanden + nicht leer: unten in der Karte einen `<ResponsiveContainer width="100%" height={32}>` mit einem minimalistischen `<LineChart>` rendern:
   - eine `<Line dataKey="value">` in `theme.primary`, `dot={false}`, `strokeWidth={2}`
   - keine Achsen, kein Grid, kein Tooltip (reine Sparkline)
   - `isAnimationActive={!prefersReducedMotion}`
4. `DashboardGrid.tsx`: den KPI-Karten `sparklineData` mitgeben. Falls die KPI-Daten keinen Verlauf liefern, einen **kleinen lokalen Demo-Verlauf** pro KPI erzeugen (stabil, z.B. aus dem KPI-Wert abgeleitet) — Hauptsache die Sparkline ist sichtbar. (In der Spec-Notiz für Luke: echte Zeitreihe → `backend-gaps.md`.)
5. Sicherstellen: Karte ohne `sparklineData` sieht weiterhin korrekt aus (Sparkline ist optional).
6. i18n: vermutlich **keine** neuen Texte nötig (rein visuell). Falls ein Label dazukommt → `berichte.*` in 4 Sprachen.
7. Verifizieren, commit, push, „fertig".

## i18n-Keys
In der Regel keine. Nur falls du ein sichtbares Label ergänzt → Präfix `berichte.`, 4 Sprachen, `{var}`-Interpolation.

## Demo-Handler
Keiner nötig. Berichte hat zwar keinen MSW-Handler, aber die Sparkline nutzt entweder vorhandene KPI-Daten oder einen lokalen Demo-Verlauf. (Hinweis für `backend-gaps.md`: echte KPI-Zeitreihen-Endpunkte fehlen.)

## Definition-of-Done
- [ ] KPI-Karten zeigen unten eine schlanke Trend-Linie (Sparkline) in der Primärfarbe.
- [ ] Sparkline nutzt `useChartTheme` (Farben aus Tokens, kein Hex) + respektiert `usePrefersReducedMotion`.
- [ ] Karte ohne `sparklineData` rendert weiterhin sauber (Prop ist optional).
- [ ] Keine Konsolenfehler, kein Layout-Bruch (Karte bleibt gleich hoch/ausgerichtet).
- [ ] Gescopter Typecheck grün, QA-Script grün, Screenshot @1440px sauber.

## QA-Hinweis
`desktop/scripts/qa-berichte-sparkline.mjs` (Vorlage aus bestehendem `qa-*.mjs`): `/#/berichte` öffnen → Dashboard-Tab → prüfen, dass in den KPI-Karten ein SVG-Pfad (Recharts-Linie, Selektor z.B. `.recharts-line`) sichtbar ist, und Raw-Key-Scan + pageErrors leer sind. Screenshot der Karten-Reihe.
