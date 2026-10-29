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

## Iteration 2 — e2e-invite-onboarding-flow — done — 2026-10-29 23:05
- commit: siehe Commit `test(e2e): cover the invite onboarding path from invitation to member login` (git log)
- gebaut: `backend/test/e2e/invite_onboarding_test.go` mit `TestInviteOnboarding` (Einladung per `POST /admin/users/invite`, Zeile `invited`, `GET /invite` mit vorbelegtem Formular + `no-store`, toter Token → 404 Fehlerzustand, `POST /invite` → „Konto angelegt" ohne Cookie/Token im HTML, Login als neuer Nutzer, `/auth/me` Name + genau Rolle member, `/admin/users/` als member 403, Roster zeigt `active`, zweiter POST und GET mit verbrauchtem Token tot, erstes Passwort funktioniert weiter) und `TestInviteResendKillsTheOldLink` (Resend → frischer Token, alter Link tot auf GET und POST und erzeugt kein Konto, neuer Link nimmt an). Rolle member wird per `GET /admin/roles` (System-Rolle, Name `member`) in ihre UUID aufgelöst. Der Flow läuft ab dem Annehmen als member; admin ist nur der Einlader.
- gate: build n.a. (nur Testdatei) | vet ok (`go vet -tags=e2e ./test/e2e/`) | lint n.a. | test n.a. | migration n.a. | rls-smoke n.a. | e2e ok (lokal, 4,985 s Testlauf, 14 PASS-Zeilen, 0 Skip/Fail; Stack ~3,5 min)
- coverage: n.a. (kein Coverage-Ziel)
- mutations-probe: 2 Proben in `internal/auth/postgres_repository.go` (CRLF-Datei, einzeilige Muster), beide rot, zurückgedreht (`git checkout`), Diff sauber: (1) `RefreshInvitationToken` überschreibt den Hash nicht → `old_link_is_dead_on_get_and_post` und `new_link_accepts` rot; (2) `AcceptInvitation` setzt `accepted_at` nicht → `second_post_with_same_token_is_rejected` (503 statt Fehlerzustand) und `get_page_after_accept_is_dead` rot.
- verify vorgaenger: sauber (Commit `45a73611`: Handler geht über `client.GetInvitationPreview`/`AcceptInvitation`, `auth.proto` + `.pb.go` im selben Commit, Seite liegt außerhalb von `/api/v1` und braucht keinen Spec-Eintrag, kein `RequirePermission` angefasst, keine neue Tabelle; Gateway-Tests hatte die Vorgängerin grün)
- neue-units: keine
- desktop-aufrufer: n.a. (nur Testcode, keine Validierungs- oder Antwortform-Änderung)
- offen: (1) Die Unit nennt das Feld `invite_token`, auf der Leitung heißt es `inviteToken` (camelCase, `adminUserResultBody` in `route_auth.go`); der Test liest `inviteToken`. Der Desktop liest laut Vorgänger-Eintrag `AdminUserResponse` unverändert, also kein Befund. (2) Der Doc-Kommentar an `adminUserResultBody` („nothing server-side dispatches invitation mail") ist seit `45a73611` veraltet — nur Kommentar, bei Gelegenheit nachziehen. (3) Der Public-Limiter hat den Flow im E2E-Env nicht gedrosselt (`RATE_LIMIT_RPS=1000`, Public-Default 10/s reichte für ~7 Requests je Flow). (4) Damit ist auch Punkt (3) aus Iteration 1 („GET /invite gegen den echten Stack") abgenommen.
