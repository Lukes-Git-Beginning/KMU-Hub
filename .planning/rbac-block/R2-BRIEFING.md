# R-2 Rollen-Baukasten — Terminal-Briefing (erstellt Session #14, 2026-07-18)

> **Für das frische Bau-Terminal.** Erst `git pull` (Stand ≥ `30ade554`). Ablauf zwingend:
> **1) Recherche-Gate → 2) gebündelte Fragen an Darien → 3) Darien-OK → 4) bauen → 5) Gates.**
> Kontext: `KONZEPT.md` (§3 Ziel-Architektur, §4 Phasenplan, §5 Entscheidungen) + `CAPABILITY-KATALOG.md`. R-1 ist komplett gebaut (siehe Datei-Karte unten) — NICHTS davon neu erfinden.

## 1. Scope R-2 (aus KONZEPT §4, alles FE mock-first)

Der IT-Baukasten im Admin-Hub. Löst die A-2-Matrix (`RolesAdminHubTab`) vollständig ab.

1. **Rollen-Liste** — 7 System-Presets + tenant-Custom-Rollen; memberCount/capabilityCount; Presets unveränderlich (nur klonbar), Limit ~20 Custom + Duplikat-Warnung.
2. **Zwei-Pane-Editor** — links Modul-Baum (30 Module, Suche), rechts pro Modul: Ebene-1-Sichtbarkeit, Ebene-2-Basis-Aktionen + Scope-Dropdown (eigene/Team/alle), Ebene-3-Fein-Schalter (Katalog). Bulk-Toggles („alle Branchen-Module aus"). Plain-Language-Zusammenfassung („kann Rechnungen sehen, aber nicht erstellen" — i18n-Bausteine `rbac.subject.*`/`rbac.action.*` existieren ×4).
3. **Klonen + based_on** — Custom = Klon eines Presets, Abweichungs-Badge „X Abweichungen vom Standard" (weclapp-Muster), Abweichungen im Editor markiert + rücksetzbar.
4. **Rollen-Vergleich** — 2 Rollen nebeneinander, Diff-Hervorhebung.
5. **„Als Rolle anzeigen"-Preview** — Shell simuliert die Rolle (Markt hat das fast nirgends). Technisch: `setDemoSessionUserId` + Auth-Store-User-Swap existieren; für Preview OHNE Account ggf. temporäre resolveCapabilities(roleId) in den permissions-Store spielen + Banner „Vorschau als X — beenden".
6. **Effektive-Rechte pro User (Admin-Sicht)** — große Version des Profil-Tabs `BerechtigungenTab` (als UI-Referenz nutzen): im User-Detail (A-1) Rollen-Chips + aufgelöste Summe mit Herkunft; Overlap-Hinweis beim Zuweisen.
7. **A-1 auf Multi-Rollen** — `AdminUser.role` (singular) → `roles: RoleId[]`; UserDetailModal-Rollen-Select → Multi-Zuweisung (Union); InviteUserDialog entsprechend. Laura (`usr-e5`) trägt im Seed schon manager+hr_admin.
8. **Guardrails-UI** — Mindestens-1-Admin-Sperre (letzten Vollzugriff-Träger nicht herabstufen/deaktivieren), Selbst-Aussperr-Warnung (eigene Rolle ändern), Eskalations-Guard (niemand vergibt Rechte über die eigenen hinaus — `admin:role:create/edit` ≠ `admin:role:assign` ist im Seed getrennt), Preset-Schutz (nicht editier-/löschbar).
9. **MSW stateful** — `POST/PATCH/DELETE /api/v1/admin/roles`, `PUT /api/v1/admin/roles/{id}/permissions`, User-Rollen-Zuweisung (`POST/DELETE /api/v1/users/{id}/roles` — BE-Routen existieren bereits in route_auth.go!). Nach Mutation: `usePermissionsStore.getState().refresh()` triggern, wenn die eigene Rolle betroffen ist (Live-Wirkung ohne Re-Login). Contract-Erweiterungen in `api/rbac-types.ts` (CreateRoleInput/UpdateRolePermissionsInput) — so schneiden, dass Lukes API sie 1:1 übernimmt (backend-gaps §RBAC).

**NICHT R-2:** Branchen-Template-Sets final (R-5) · Audit-Log-UI (R-5) · Zentria-Setup-Zugang (R-5) · Modul-Aktions-Gating in den Modulen (R-3) · Objekt-Grants (Phase 2).

## 2. Recherche-Gate R-2 (VOR den Fragen, KONZEPT §4-Kasten)

**Auftrag:** Rollen-Editor-UIs der Marktführer **visuell** ansehen — Screens/Layout/Interaktion UND Funktionsumfang (beide Achsen, memory `feedback_market_driven_workflow`). Web-Research-Agent(s) mit Bild-/Doku-Quellen:
- **Zoho CRM** Profiles-Editor (Modul×Aktions-Matrix, Klonen) + Roles/Hierarchie getrennt
- **weclapp** Rollen/Rechte (based_on/Abweichungen)
- **Odoo** Access Groups (Vererbung — als Negativ-/Komplexitäts-Beispiel)
- **monday.com** Custom Roles Editor (2024/25, cleanes Toggle-UI)
- **Microsoft Entra** Custom-Role-Wizard (Suche/Kategorien in großen Permission-Listen)
- **Personio** Zugriffsrollen-Editor (Datenkategorie × Zugriffsebene × Scope — HR-Referenz)
Fokusfragen: Wie strukturieren sie 100+ Schalter ohne Überforderung (Baum? Tabs? Suche? Presets?), wie zeigen sie Abweichungen/Vererbung, wie sieht Vergleich/Preview aus, was ist der Save-Flow (sofort vs. Entwurf), wie warnen sie bei Guardrail-Verstößen. Ergebnis → Feature-/Layout-Entscheidungsvorlage mit 2–3 Optionen für die gebündelten Fragen.

**Gebündelte Fragen an Darien danach** (erwartbare Themen): Editor-Layout-Variante (Zwei-Pane vs. Matrix vs. Wizard), Save-Verhalten (live vs. Entwurf+Übernehmen), Ort der Effektive-Rechte-Ansicht (User-Detail vs. eigener Tab), Preview-Mechanik (Session-Swap vs. Overlay), was mit der A-2-Matrix-Route passiert (Redirect?).

## 3. Datei-Karte nach R-1 (Basis — nicht neu bauen)

| Bereich | Datei(en) |
|---|---|
| Contract | `desktop/src/renderer/src/api/rbac-types.ts` (+ `rbac-client.ts`) — Role/CapabilityGrant/EffectivePermissions, SCOPE_ORDER |
| Seeds/Presets | `mocks/data/rbac.ts` — `ROLE_DEFS` (Grants = SSOT), `seedRoles()`, `USER_ROLE_ASSIGNMENTS`, `resolveCapabilities()`, `DEMO_PROFILES`, `get/setDemoSessionUserId` — ⚠ `git add -f` (gitignore `data/`) |
| Handler | `mocks/handlers/rbac.ts` (GET me/permissions + GET admin/roles) — R-2 macht CRUD stateful |
| Store/Hook | `stores/permissions.ts` (persist + Auth-Subscription + Client-Fallback) · `hooks/useCapability.ts` |
| Mappings | `config/capabilities.ts` (NAV_ITEM_MODULE, SETTINGS_*, TEAM_TAB_CAPABILITY, MODULE_KEYS) · `config/roles.ts` (RoleId-7er, roleLabelKey/roleDescriptionKey) |
| UI-Referenz | `modules/profil/tabs/BerechtigungenTab.tsx` (Effektive-Rechte-Darstellung, Scope-/Herkunfts-Badges) |
| Abzulösen/Umzubauen | `modules/admin/roles/RolesAdminHubTab.tsx` (A-2) · `modules/admin/users/*` (A-1 multi-role) · `api/admin-types.ts` (PermissionMatrix legacy) · `mocks/data/admin-permissions.ts` (legacy markiert) |
| Rollen-Optik | `modules/admin/users/presentation.ts` (ROLE_DOT/ROLE_ORDER — statische Tailwind-Klassen!) |
| i18n | `rbac.*`-Namespace ×4 komplett (roles/module/subject/action/scope/effective) — Einfüge-Script-Muster `scripts/i18n-rbac-r1.mjs` |
| Gates | `tsconfig.rbaccheck.json` (Muster für scoped check) · `scripts/qa-rbac-fundament.mjs` (QA-Muster: STUB/ONB/NOLAUNCH, waitForText, Switcher-Helpers rechts unten) |

## 4. Bekannte Stolpersteine (aus R-1 gelernt)

- **Hydrator-Race:** Permissions laden via Auth-Store-Subscription — beim Editieren von Rollen im Baukasten nach Save `refresh()` aufrufen, sonst wirkt die Änderung erst beim nächsten User-Wechsel.
- **ProfileSwitcher liegt unten RECHTS** (fixed bottom-4 right-4) — QA-Klickpunkte entsprechend; Panel schließt nur per Backdrop/X, nicht per Esc.
- **Demo-Dashboard zeigt Rollen-fremde Karten/Banner** (Ebene 2, erst R-3) — im QA nicht als Fehler werten.
- **„Infrastruktur"/„Dokumente" fehlen in frischer Nav** durch vorbestehenden Optional-Module-/Business-Profil-Filter — kein RBAC-Bug.
- Standard-Gates: i18n ×4 (`{var}`, ICU-Plural), gescopter tsc (Full-tsc nie), `eslint src/ --quiet` vor Push (fand in R-1 einen Hook-in-`&&`-Bug), Screenshot-QA + **Bilder ansehen**, 1 Dev-Server, 1 Commit + Push (= Auto-Deploy auf Hetzner, cd.yml scharf).

## 5. Nach R-2

R-3 Enforcement-Sweep (batch-weise à ~5 Module, Katalog je Modul gegen UI kuratieren) → R-4 HR-Seite → R-5 Audit/Zentria-Zugang/Branchen-Sets. Onboarding O-0 erst nach RBAC-Review (Darien-Entscheid #13).
