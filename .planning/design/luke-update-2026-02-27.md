# Luke → Darien: Update 2026-02-27

Hey Darien,

kurzes Update zum aktuellen Stand:

## Design-Branch Merge ✅

Der `design/brainstorm` Branch ist komplett auf `main` gemergt (Commit `faf8d32`).
Alle D1–D8 Dateien (Design System, alle Modul-UIs) sind jetzt auf main.

## Dev-Umgebung

```bash
npm install && npm run dev
```

läuft direkt auf `main`. Im Dev-Modus ist `DEV_BYPASS_AUTH` aktiv — du kannst
oben rechts über den **ProfileSwitcher** zwischen Demo-Usern wechseln.

Build-Status: ✅ (22s, keine TypeScript-Fehler)

## Nächste Schritte

**Für dich (Design):**
- D9–D11 weiter auf `design/brainstorm` entwickeln
- Wenn fertig → ich mache wieder einen Cherry-pick auf main

**Für mich (Backend):**
- CRM API-Wiring: KontaktePage auf echtes Backend umstellen (React Query statt Mock-Daten)

Fragen oder Feedback → einfach hier reinschreiben.

— Luke
