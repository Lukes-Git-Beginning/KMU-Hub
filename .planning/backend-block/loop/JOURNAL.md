# Backend-Nachtloop — Journal Lauf 18

Append-only. Ein Eintrag je Iteration, **ans Dateiende**, nie einsortieren. Form und Pflichtzeilen
stehen in `ITERATION.md` Schritt 6.

Frühere Läufe liegen vollständig im Archiv:
`archive/lauf-1-2/` (58 Units) · `archive/lauf-3/` (61) · `archive/lauf-4/` (54) ·
`archive/lauf-5/` (41) · `archive/lauf-6/` (46) · `archive/lauf-7/` (71) ·
`archive/lauf-8/` (94) · `archive/lauf-9/` (37) · `archive/lauf-10/` (93) ·
`archive/lauf-11/` (131 Einträge, davon 121 done) · `archive/lauf-12/` (99, davon 41 done) ·
`archive/lauf-13/` (120, davon 21 done — abgestürzt) · `archive/lauf-14/` (110, alle done —
Backlog leergelaufen, gemergt als `cadf4bf4`) · `archive/lauf-15/` (74, alle done — Backlog
leergelaufen, PR #27) · `archive/lauf-16/` (82, alle done — Backlog leergelaufen, PR #28) ·
`archive/lauf-17/` (132 Einträge, 116 done in 117 Iterationen — Deadline 10:00, PR #29).
Alle inkl. Laufbilanz.

---

## Laufkontext

- **Ausgangspunkt:** Lauf 17 gemergt als `90b03133` (PR #29) und deployt (CD 2026-10-26 09:49,
  `/health` meldet `90b03133`). `backend-loop` = `main` plus Staging-Commits (Archiv Lauf 17,
  dieser Lauf) plus die FE-Commits vom 2026-10-29.
- **Migrationen:** Repo-Kopf **364**. Nummer immer zur Laufzeit ermitteln:
  `ls backend/migrations | grep -E "^[0-9]{6}" | sort | tail -1`
- **Coverage-Start:** **76,8 %** gesamt (Run 40300605217 auf `90b03133`). Paketwerte je Unit.
- **Queue:** 45 Units, roter Faden „Fehlerklassen-Zwillinge“. Pilot-Pfad und Entschiedenes (11, inkl.
  Berichte-Token als Hash, seit der Desktop-Teil `4e5c830d` gemergt ist),
  Einzelfälle aus Lauf 17 (14), Zwillings-Scans (11 plus 2 Reviews), Coverage-Auffüller (7).
  Begründung, Regeln und Sperren im Kopf von `BACKLOG.yml`.
- **Neu:** Scans legen ihre Fix-Units direkt hinter sich (Regel 1). `IST:`-Subtests aus Lauf 17
  werden zu Erwartungen, sobald ihr Fix gebaut ist (Regel 3).

## Entscheidungsrunde 2026-10-29 (Luke, vier Runden) — Ergebnis

Je Unit steht die Entscheidung im Feld `decision` von `BACKLOG.yml`. Kurzfassung:

- **Einladung:** Mail über den vorhandenen System-Mailer, Annahmeseite `/invite` im Gateway nach dem
  Muster `/reset-password`. Kein Deep-Link in die App, kein neues Env.
- **`notification.send`:** Zustellung über den Event-Bus an den vorhandenen `InboxConsumer`, keine
  neue RPC. Automation darf auf das eigene Event nicht wieder triggern.
- **Aufnahme:** Teilnehmer willigen in die `pending`-Aufnahme ein, Egress erst nach allen Ja und dem
  Initiator. Der Status geht als String über den Wire.
- **Abo-Rechnung:** „Jetzt erzeugen“ vor Fälligkeit ergibt 409. Vorziehen gibt es nicht.
- **Rechte:** manager liest Fuhrpark-Schäden mit Scope all. Geteilte E-Mail-Vorlage anlegen nur mit
  dem groben Schreib-Key, member bekommt 403.
- **GPS bei Privatfahrten:** so lassen, `lean:`-Marker gesetzt (Staging-Commit), Trigger „erster
  Kunde mit Tracker“.
- **Retention:** Fristen bleiben bis zur anwaltlichen Beratung liegen. Der Scan prüft nur, ob
  vorhandene Löschwege die Kindtabellen mitnehmen.
- **Produktion-Hülle:** Der Desktop entpackt (FE, tagsüber erledigt).
- **Tagsüber, nicht im Lauf:** vier FE-Units (Berichte-Link einmalig zeigen, Einladungsdialog,
  Gutschrift-Storno über `/cancel`, Produktion entpacken), DSGVO-Bereinigung der Prod-Testkonten,
  Darien-Briefing zu Schichten.

## Was aus Lauf 17 gelernt wurde — drei Dinge, die dich direkt betreffen

1. **Die teuersten Funde kamen nebenbei.** `work.create_task` ohne `created_by` (Totalausfall) und
   der `notification.send`-Stub fielen erst in der vorletzten Coverage-Unit auf. Deshalb scannt
   dieser Lauf die Fehlerklassen gezielt, statt auf Zufallsfunde zu warten.
2. **RLS verdeckt fehlende Tenant-Prädikate.** In sechs Coverage-Units überlebte die Mutationsprobe
   auf das Tenant-Prädikat zuerst, weil RLS die Zeile schon verbarg. Erst ein Subtest unter
   System-Kontext machte sie rot (Regel h).
3. **CRLF-Altdateien verfälschen Proben.** Mehrzeilige Probenmuster trafen in CRLF-Dateien nicht
   und galten zuerst als „überlebt“. Zeilenende vor der Probe prüfen.

---
## Iteration 1 — feat-auth-invite-mail-and-accept-page — done — 2026-10-29 22:40
- commit: siehe Commit `feat(auth): mail the invitation link and accept it on a gateway /invite page` (git log)
- gebaut: `Mailer.SendInvitationEmail` + `systemMailer` (deutsche Mail, Gültigkeit aus `auth.InvitationExpiry`, Token nie geloggt); Versand in `createInvitation` (deckt `CreateInvitation` und `InviteAdminUser`) und in `ResendAdminUserInvite`, Link `<Host aus PasswordResetBaseURL>/invite?token=`, kein neues Env. Versandfehler: `slog.Error` mit `invitation_id`, Einladung bleibt. Neue RPC `GetInvitationPreview` (Proto + Regen) für das Vorbelegen/Erkennen toter Links. Seite `internal/gateway/invite_page.go`+`.html` (GET/HEAD/POST `/invite` hinter dem Public-Limiter, kein JS, CSP/no-store/noindex/no-referrer, Token im Hidden-Field, keine Session, TokenPair wird verworfen).
- gebaut (Zusatz, Root Cause): `auth.PasswordValidator` nimmt jetzt den Tenant als Parameter (`passwordPolicyAdapter` in `cmd/auth`). Vorher las `security/password` den Tenant aus ctx, den der Pre-JWT-Reset nie hatte — die Policy-Prüfung beim Reset schlug IMMER fehl und wurde übersprungen. Jetzt gilt sie beim Reset und neu bei der Einladungsannahme.
- gate: build ok | vet ok | lint ok (0 issues) | test ok (internal/auth 323 RUN/0 SKIP, internal/gateway 0 SKIP inkl. TestOpenAPIRouteDrift, internal/server 0 SKIP, cmd/auth ok) | migration n.a. | rls-smoke n.a. | e2e ok (lokal, 237 s, `TestInvitationFlow` 6/6)
- coverage: internal/auth 77,9 % -> 78,5 %
- mutations-probe: 7 Proben, alle rot und zurückgedreht, Diff sauber: Mail in createInvitation entfernt (4 Tests rot inkl. DB), Mail im Resend entfernt (`TestInviteAdminUser_DB_MailsWorkingSingleUseLink`), Policy-Prüfung in AcceptInvitation entfernt (`TestAcceptInvitation_EnforcesTenantPasswordPolicy`), Link ins Fehler-Log (`..._MailFailureKeepsInvitationAndIsLogged`), verbrauchtes Token nicht terminal (`TestInvitePage_PostCreatesAccountOnceWithoutSession`), `no-store` getauscht (`TestInvitePage_SecurityHeadersAndHead`), Preview-RPC ohne Service (`TestAuthGRPC_GetInvitationPreview`). Mehrzeilige Proben erst nach CRLF-Anpassung getroffen (Regel h).
- verify vorgaenger: n.a. (erste Iteration des Laufs, Vorgänger `b9fd78da` ist ein Doku-Commit)
- neue-units: keine
- desktop-aufrufer: `POST /api/v1/admin/users/invite` -> `api/hooks/useAdminUsers.ts` -> passt (Antwort `AdminUserResponse`/`invite_token` unverändert); `POST /api/v1/invitations/{token}/accept` (antwortet jetzt 400 bei Policy-Verstoß) -> nur `api/types.ts`, kein Aufrufer -> passt
- offen: (1) Verhaltensänderung: Passwort-Reset prüft jetzt real gegen die Tenant-Policy (Default min. 12 Zeichen, 50 Bit Entropie) — der Hinweis der Reset-Seite sagt jetzt „Mindestens 8 Zeichen — dein Unternehmen kann längere verlangen“, ein Verstoß kommt mit der allgemeinen Policy-Meldung zurück. (2) `ProvisionTenant` (erste Admin-Einladung eines neuen Tenants) verschickt weiterhin KEINE Mail — nicht im Scope, Token kommt wie bisher in der Antwort. (3) `GET /invite` gegen den echten Stack prüft erst `e2e-invite-onboarding-flow`. (4) Proto `auth.proto` geändert und regeneriert (protoc 33.4, gen-go 1.36.11, gen-go-grpc 1.6.1 wie bisher).
