# Review-Fäden — <Modul>

> Eine Datei pro Modul. **Jeder Bau-Strom trägt nach jeder fertigen Phase einen Eintrag ein.** Das ist die Vorlage für den gemeinsamen Feinschliff-Review (Darien navigiert, Team schaut — wie beim Profil-Fenster). Ziel: Darien/Reviewer klickt den Pfad nach, schaut auf die „Worauf achten"-Punkte, hakt ab oder gibt Feinschliff-Feedback.

**Modul:** `<modul>` · **Strom:** <N/D/L> · **Reviewer (zugeteilt):** <offen / Name>

---

## Phase <N> — <Titel>  ·  <commit-sha>  ·  Status: ⬜ ungereviewt

**Hinklicken (Pfad in der App):**
- Route(n): `/<modul>/…`
- Schritte: Sidebar → <Modul> → … (genaue Klickfolge, wie in der Karte-Klick-Checkliste)

**Worauf achten (Feinschliff):**
- [ ] Layout/Hierarchie bei voller Breite + schmal
- [ ] Keine Raw-i18n-Keys, keine Emojis, keine ASCII-Umlaute
- [ ] Leere Zustände sinnvoll (keine leeren Screens)
- [ ] Interaktionen echt (nicht nur sichtbar): <z.B. Drag, Filter, Speichern>
- [ ] <modul-spezifischer Punkt>

**Screenshots:** `desktop/.qa-screenshots/<modul>-*.png` (welche QA-Läufe)

**Bekannte offene Punkte / Backend-Bedarf:**
- <z.B. „Demo-Daten dünn (Luke), Auslastung Mock, …">

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

<!-- Nächste Phase hier anhängen, gleiche Struktur. Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
