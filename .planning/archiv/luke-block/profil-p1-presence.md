# Phase (Pilot, profil) — Profil: Presence-Status + Avatar-Upload-UI

> **Modul:** profil · **Risiko:** niedrig · **Backend:** Avatar-Upload-Endpoint + echte Presence = Luke (du baust beides eh) → bis dahin **mock-first**.
> Erster FE-Pilot der profil-Lane (nach dashboard). profil ist die persönliche Seite — kein klassisches „Modul-Settings-Panel", sondern Feature-Ausbau am Profil selbst.

## Ziel
Im Profil zwei sichtbare Verbesserungen: (1) **Presence-Status** setzen (Online/Abwesend/Beschäftigt/Unsichtbar) inkl. Anzeige, (2) **Avatar-Upload-UI** (Datei wählen + Vorschau/Cropping-Platzhalter), mock-first bis der Upload-Endpoint steht.

## Ist-Stand (was schon da ist)
- `modules/profil/ProfilPage.tsx` + `modules/profil/tabs/` (`ProfilTab`, `AbwesenheitenTab`, `DokumenteTab`, `ZeiterfassungTab`). Store `stores/profile.ts`.
- Presence wird modulweit genutzt (z.B. team/kommunikation zeigen Status-Punkte) — prüfen, ob es einen `presence`-Store/Feld gibt (grep `presence`, `myStatus`) und daran andocken statt neu erfinden.

## Muster-Vorlagen
- Status-Picker: kleines Popover mit farbigem Punkt je Status (Muster: Status-Dropdowns in team/kommunikation).
- Avatar mit Kamera-Button: der bestehende Camera-Button im Profil (wartet laut `backend-handover-luke.md` auf Upload) — die UI dranbauen (Datei-Input + lokale Vorschau via `URL.createObjectURL`), echter Upload als TODO markiert.

## Schritte
1. `marathon/luke-fe`. App `/profil` öffnen.
2. **Presence:** an vorhandenen presence-Store andocken (oder `stores/profile.ts` um `presenceStatus` erweitern). Status-Picker im Profil-Header; gewählter Status persistiert + spiegelt sich im Avatar-Punkt.
3. **Avatar-Upload-UI:** Datei-Input hinter dem Kamera-Button, lokale Bild-Vorschau, „Speichern" mock-first (echter `POST`-Upload → `backend-handover-luke.md`). Kein Bruch, wenn kein Bild gewählt.
4. **i18n ×4** `profil.presence.*` / `profil.avatar.*`, einfache Klammern.
5. Verifizieren, Review-Faden, commit, push.

## Definition-of-Done
- [ ] Presence-Status im Profil setzbar, persistiert, spiegelt sich im Avatar-Punkt (mind. lokal/mock).
- [ ] Avatar-Upload-UI: Datei wählen + Vorschau; „Speichern" mock-first, sauber als Vorschau markiert.
- [ ] Kein Bruch bestehender Profil-Tabs. i18n ×4, scoped tsc grün (`tsconfig.profil-presence.json`), QA grün, Screenshots angesehen.
- [ ] Review-Faden in `reviews/profil.md` (echter Upload/Presence = Backend-Bedarf vermerkt).

## QA-Hinweis
`scripts/qa-profil-presence.mjs`: `/#/profil` öffnen → Status wechseln → Avatar-Punkt ändert Farbe → Datei-Input sichtbar → 0 Roh-Keys/pageErrors. Screenshots Header + Avatar-UI.
