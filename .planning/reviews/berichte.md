# Review-Fäden — berichte

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `berichte` · **Strom:** N · **Reviewer (zugeteilt):** offen

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->

## 🟡 Vor-Review durch Strom D — Nico Pilot 02: KPI-Sparklines (2026-06-10)

> Nicos Review-Gate (BACKLOG `nico-block`) wartet auf Darien. Strom D hat vorgeprüft, damit das Gate schnell geht. **Pilot 01 (notifications-Ruhezeiten, `279dee2`) ist bereits in main gemergt — nur Pilot 02 (`marathon/nico`, 3 Commits) ist offen.**

**Code-Review (Diff gelesen): sauber.**
- Deterministischer Verlauf (Seed aus KPI-id, kein `Math.random` pro Render) — Linie stabil über Re-Renders
- Trend-Richtung folgt `change_percent` → Linie passt visuell zur Badge
- `usePrefersReducedMotion` + `useChartTheme` korrekt genutzt, optionaler Prop (kein Breaking Change), `aria-hidden`
- ASCII-Umlaut-Fix (`7dcbb4b`) sauber nachgezogen

**Visuell verifiziert (eigener Dev-Server auf seinem Branch, Screenshot `nico-review-berichte.png`):**
- 9/9 KPI-Karten mit Sparkline, QA-Script grün (rawKeys [], pageErrors [])

**Feinschliff-Punkte für Dariens Review (Vorschläge, keine Blocker):**
1. 🟡 **Linien wirken fast flach** — Noise/Amplitude klein bei 32px Höhe; bei „+12.4%" sieht man kaum Steigung. Option: Serie auf min–max normalisieren, damit der Trend sichtbar wird.
2. 🟡 **Linienfarbe immer Primärgrün** — auch wenn die Badge rot ist (z.B. „Offene Rechnungen −8.1%"). Option: Linie folgt der Goodness-Farbe der Badge. Designentscheidung Darien.
3. ℹ `buildSparklineSeries` läuft ohne `useMemo` bei jedem Grid-Render (deterministisch, 9 Karten — unkritisch, aber billig zu memoisieren).
4. ℹ Vorbestehend (nicht Nicos Phase): die großen Charts darunter („Umsatzverlauf", „Tickets nach Priorität") zeigen im Demo „Noch keine Daten geladen" — Demo-Handler-Lücke für die Folgephase.

**Merge-Lage:** Konflikte mit anderen Strömen nur additiv (de.json, backend-gaps) — trivial.

**Reviewer-Notizen (Darien):**
- _…_
