# Cosmi Modul-Ist-Abgleich

> Realer Code-Stand pro Markt-Feature (Gegenstück zur leeren COSMI-Spalte in `cosmi-modul-marktvergleich.txt`).
> Grundlage für Feature-Parität bis Launch 01.09.2026. Erfasst ab 2026-06-01.

## Legende
- **FE**: ✓ voll umgesetzt · ◐ teilweise/Stub/Mock · ✗ fehlt
- **BE**: ✓ Endpoint+Service nutzbar · ◐ teilweise · ✗ fehlt
- **Aktion**: 🟢 fertig · 🟡 Frontend-Arbeit (BE fertig, nur anklemmen — **mein Job, kein Luke**) · 🔴 Backend-Arbeit (→ Luke) · 🟠 beides fehlt

## Hauptbefund Welle 1
Häufigstes Muster: **Backend + API-Hooks fertig, aber Frontend hängt an Mock/lokalem State statt an der echten API.** Viel „fehlende Funktion" = reine FE-Anbindung (kein Neubau, kein Warten auf Luke).

---

## Entscheidungen & Duplikat-Klärungen

| Thema | Befund | Entscheidung |
|---|---|---|
| `calendar` vs. `kalender` | `calendar/` = leerer Stub (Altlast); `kalender/` = echt (3064 Z., volle Views/Hooks) | **`calendar/` löschen**, `kalender` ist kanonisch |
| `chat` vs. `kommunikation` | Beide aktiv: `chat` = internes Team-Chat, `kommunikation` = externer Unified Inbox (Mail/WhatsApp/Widget) | **ZUSAMMENFÜHREN zu EINEM Modul unter `kommunikation`** (Darien-Entscheidung 2026-06-01). Internes Team-Chat + externe Kanäle in einem Modul. Dazu nötig: **Modul-Einstellung, um die Verknüpfungen für den nicht-internen (externen) Chat zu konfigurieren & verwaltbar zu machen** (Kanäle anbinden/verwalten). |
| `video` vs. `meetings` | — | klärt Welle 2 |

---

## Welle 1 — ZFA-Must-Module

### crm
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Kontakt- & Firmenverwaltung | ✓ | ✓ | 🟢 | crm/contacts/, route_crm.go |
| Visuelle Deal-Pipeline (D&D) | ◐ | ✓ | 🟡 | DealPipelineView.tsx — Kanban da, **kein Drag&Drop** (dnd-kit fehlt) |
| Aktivitäten & Aufgaben am Kontakt | ✓ | ✓ | 🟢 | route_crm_activities.go |
| E-Mail-Integration / 2-Wege-Sync | ◐ | ✓ | 🟡 | BE-Link-API da; FE nur mailto, E-Mail-Thread am Kontakt einbinden (emailLinkApi) |
| Lead-Scoring | ✗ | ✗ | 🟠 | komplett fehlend (FE+BE) |
| Angebote/Quotes aus Deal | ◐ | ✓ | 🟡 | Button da; Quote-Übersicht im DealDetail fehlt |
| Umsatz-Forecasting | ✗ | ◐ | 🔴 | weighted_value im Pipeline-Report; dedizierter Forecast-Endpoint + Widget fehlen |
| E-Mail-Marketing / Kampagnen | ◐ | ✗ | 🟠 | NewsletterPanel.tsx = Mock; Kampagnen-Service fehlt komplett |
| Workflow-Automatisierung | ✓ | ✓ | 🟢 | automatisierung-Modul voll, route_automation.go |
| Mobile App | ✗ | ✗ | 🟠 | Electron-Desktop only → PWA-Architekturentscheidung |
| Pipeline-/Conversion-Reporting | ◐ | ✓ | 🟡 | BE-Reports da; CRM-Report-Seite + Hooks im FE fehlen |

Fehlend (Vorschlag): Kontakt-Merge-Button im CRM (BE da), Consent-Tab im CRM-Detail (BE da), Follow-up-Reminder.
Auffällig: `ContactTimeline.tsx` existiert, **nicht eingebunden**. Tag-Mutation gibt 501 (nur gRPC).

### kontakte
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Zentrale Kontaktdatenbank | ✓ | ✓ | 🟢 | KontaktePage.tsx, route_crm.go |
| Import (CSV/Excel) | ◐ | ◐ | 🔴 | FE parst CSV lokal (Hook ungenutzt); **XLSX-Endpoint fehlt** |
| Tags & Segmentierung | ◐ | ✓ | 🟡 | GroupManagerDialog lokal (Zustand); /api/v1/tags ungenutzt |
| Dubletten-Erkennung | ◐ | ✓ | 🟡 | FE matcht clientseitig; BE FindDuplicates+Merge ungenutzt |
| Kontakthistorie/Timeline | ◐ | ✓ | 🟡 | FE rendert immer leer; useContactTimeline ungenutzt |
| Custom Fields | ◐ | ✓ | 🟡 | FE lokaler State; /api/v1/custom-fields ungenutzt (geht bei Neustart verloren) |

Fehlend: Consent-Panel im Kontakte-Modul (existiert, nicht eingebunden), Bulk-Aktionen, `referred_by` (Empfehlungs-Tracking, ZFA-relevant).
Auffällig: kontakte/ und crm/contacts/ = parallele Implementierungen derselben Daten, inkonsistent (lokal vs. API).

### dialer
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Click-to-Dial | ✓ | ✓ | 🟢 | DialerWorkspacePage.tsx, route_dialer.go |
| Predictive / Auto-Dialer | ◐ | ◐ | 🔴 | bewusst Phase 3 (disabled) |
| Kampagnen- & Kontaktlisten | ✓ | ✓ | 🟢 | vollständig |
| Gesprächsaufzeichnung | ✗ | ✗ | 🟠 | recording_url fehlt; strukturell an Video-Infra gekoppelt |
| Agent-Dashboard & Statistiken | ✓ | ✓ | 🟢 | AgentDashboardPage.tsx |
| CRM-Verknüpfung (CTI) | ✓ | ✓ | 🟢 | dialer/crm_bridge.go |
| AMD (Anrufbeantworter-Erkennung) | ✗ | ✗ | 🟠 | komplett fehlend |

Fehlend: Callback-Listenview (BE unterstützt callback_at), Gesprächsskript-Engine (ZFA-relevant), DNC-Liste-UI.
⚠ **DSGVO-Risiko:** `consentAsserter` ist im Standard-Konstruktor `nil` — nur `NewServiceWithConsent` verdrahtet den Check. Falls Standard-Konstruktor genutzt wird, werden Einwilligungs-Prüfungen umgangen.

### kalender (`kalender/`, BE caldav + route_calendar.go)
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Termine & Events | ✓ | ✓ | 🟢 | Day/Week/Month + D&D |
| Mehrere/geteilte Kalender | ✓ | ✓ | 🟢 | Subscribe/Browse |
| Wiederholungen (RRULE) | ◐ | ✓ | 🟡 | BE da; FE-RRULE-Builder fehlt |
| Erinnerungen | ◐ | ✓ | 🟡 | BE da; useSetEventReminders nicht im Formular |
| Ressourcen-Buchung (Räume) | ◐ | ✓ | 🟡 | FE nutzt hardcodierte ROOMS statt /resources |
| iCal/CalDAV Sync | ✓ | ✓ | 🟢 | caldav_backend.go, CalDAVSettingsTab |
| **Verfügbarkeit/Terminbuchungs-Link** | ◐ | ✗ | 🔴 | **FE komplett Mock; BE fehlt ganz — KRITISCH für ZFA-Akquise** |
| Einladungen & RSVP | ✓ | ✓ | 🟢 | RSVP-Buttons + BE |

Fehlend: öffentl. Booking-Link + Slug + Availability (s. Terminbuchung), Team-Verfügbarkeits-Overlay, Feiertags-Seed-UI (BE da), Timezone-Auswahl.
Auffällig: ROOMS/TEAM_MEMBERS/BOOKING_SERVICES/MOCK_BOOKINGS alle hardcodiert. Route-Diskrepanz `/calendars` vs `/calendar`.

### dokumente (BE document)
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Datei-Upload & Ordnerstruktur | ✓ | ✓ | 🟢 | DokumentePage.tsx |
| Versionierung & Wiederherstellung | ✓ | ✓ | 🟢 | VersionHistoryPanel.tsx |
| Granulare Freigaben/Rechte | ◐ | ✓ | 🟡 | nur read/write; ShareDialog da |
| Online-Bearbeitung (WOPI/Office) | ✓ | ✓ | 🟢 | OnlyOfficeEditor.tsx, route_wopi.go |
| Volltextsuche | ✓ | ✓ | 🟢 | HandleSearchFiles |
| Kommentare | ✗ | ✗ | 🟠 | Icon importiert, kein UI/BE |
| Externe Share-Links | ◐ | ✗ | 🔴 | ShareLinkDialog = Mock; BE Token-Store fehlt |
| Self-hosted/EU-Daten | ✓ | ✓ | 🟢 | MinIO, self-hosted |

Fehlend: Datei-Kommentare, echte Share-Links, Signatur-Workflow (Verknüpfung vertraege), Bulk-Ops, Favoriten-Tab (BE-Flag da).

### formulare (BE formulare)
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Drag&Drop Form-Builder | ◐ | ✓ | 🟡 | Store reorder da; **DnD-Interaktion fehlt** (dnd-kit) |
| Bedingte Logik (Show/Hide) | ✓ | ✓ | 🟢 | ConditionalLogic |
| Submissions-Verwaltung | ✓ | ✓ | 🟢 | Tab eingänge |
| Webhooks | ✓ | ✓ | 🟢 | Webhook-CRUD + Deliveries |
| Vorlagen | ✓ | ✓ | 🟢 | IsTemplate + Duplicate |
| DSGVO-konform (EU-Hosting) | ◐ | ◐ | 🟡 | self-hosted; Consent-Feldtyp + IP-Opt-in fehlen |
| Mehrseitige Formulare | ✓ | ✓ | 🟢 | PageCount |

Fehlend: öffentlicher Submit-Endpoint (IsPublic-Flag da, kein public Endpoint), Submission-Mail-Benachrichtigung, File-Upload-Feldtyp-Handling.

### vertraege (BE vertraege)
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Vertrags-Ablage | ◐ | ✓ | 🟡 | **ganze Seite auf Zustand-Store**; Hooks (useContracts…) ungenutzt |
| Laufzeiten & Fristen | ◐ | ✓ | 🟡 | wie oben (FE-Datenbindung) |
| Kündigungs-Erinnerungen | ◐ | ✓ | 🟡 | BE Reminder-CRUD da; FE-Hooks ungenutzt |
| Versionen & Anhänge | ◐ | ◐ | 🟡 | UploadDocument = Stub (TODO MinIO); FE MOCK_DOCUMENTS |
| Digitale Signatur | ◐ | ◐ | 🟡 | nur DB-Feld (Phase D Skribble); ESignaturDialog liest Mock |
| Audit-Log/Nachverfolgung | ✗ | ✗ | 🟠 | keine History-Tabelle |

Hauptgap: **gesamte Seite läuft gegen Mock-Store, fertige Hooks ungenutzt** — größter Einzel-FE-Gap, aber reine FE-Arbeit.

### mails (BE email)
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| IMAP/SMTP Anbindung | ✓ | ✓ | 🟢 | MailServerSettingsTab |
| Mehrere Konten / Unified Inbox | ◐ | ✗ | 🔴 | nur 1 Account/User; ListAccounts fehlt |
| Signaturen (pro Konto) | ✓ | ✓ | 🟢 | Signature-CRUD |
| Vorlagen/Quicktext | ◐ | ✗ | 🔴 | FE hardcodiert; Template-CRUD fehlt |
| Regeln & Filter | ✗ | ✗ | 🟠 | fehlt komplett |
| Volltextsuche | ✓ | ✓ | 🟢 | tsquery |
| Exchange/EWS Support | ✗ | ✗ | 🟠 | fehlt |
| Verschlüsselung (PGP/S-MIME) | ✗ | ✗ | 🟠 | fehlt |

Positiv/differenzierend: DSGVO-Aufbewahrungsfristen-Anzeige schon da (ZFA-relevant).

### kommunikation (chat + inbox) — *werden zu einem Modul zusammengeführt*
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Channels (öffentlich/privat) | ✓ | ✓ | 🟢 | chat/channels, route_chat.go |
| Direktnachrichten (DMs) | ✓ | ✓ | 🟢 | useDMs |
| Threads | ✓ | ✓ | 🟢 | ThreadPanel |
| Reaktionen & @Mentions | ◐ | ◐ | 🟠 | Reaktionen lokal; AddReaction-Endpoint fehlt; ⚠ useReactions.ts importiert falsch aus video-client |
| Datei-Sharing im Chat | ◐ | ◐ | 🟠 | Upload()-Service da, **Upload-Route fehlt** in route_chat.go |
| Volltextsuche im Verlauf | ✓ | ✓ | 🟢 | /chat/search |
| Audio-/Video-Call integriert | ✗ | ◐ | 🔴 | Video als eigenes Modul; „Call starten" im Chat fehlt |
| Unified Inbox (kanalübergreifend) | ✓ | ✓ | 🟢 | kommunikation/, route_inbox.go (email/chat/notification, RoutingRules) |
| Bots / Integrationen | ✗ | ✗ | 🟠 | fehlt |

Merge-Aufgabe: chat + kommunikation in ein Modul; **Einstellung für externe Kanal-Verknüpfungen** (verwaltbar). Routing-Rules-Infra im inbox-Service vorhanden.

### helpdesk (BE helpdesk, hinter featureflag modules.helpdesk)
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Ticket-Verwaltung | ◐ | ✓ | 🟡 | **FE 100% Mock-Store**; useTickets ungenutzt |
| SLA / Eskalation | ◐ | ✓ | 🟡 | SLABadge aus Mock; BE SLA-CRUD da |
| Canned Responses | ◐ | ✓ | 🟡 | aus Mock; BE-CRUD da |
| Ticket-Merge | ✗ | ✓ | 🟡 | BE HandleMergeTickets; FE-UI fehlt |
| Ticket-Zeiterfassung | ✗ | ✗ | 🟠 | kein time_spent-Feld |
| Kunden-/Org-Zuordnung | ◐ | ✗ | 🔴 | nur Textfeld; contact_id/org_id im Model fehlt |
| Multi-Channel (Mail/Chat/Tel) | ✗ | ✗ | 🟠 | kein source_channel; Inbox-Adapter könnten Tickets erzeugen |
| Self-hosted/EU-Daten | ✓ | ✓ | 🟢 | Tenant-Isolation getestet |

Hauptgap: **komplette FE-Backend-Trennung** — Backend + Hooks fertig, FE nutzt nur Mock-Store. Größte Einzellücke, aber FE-only.
Fehlend: Ticket-Tags, CSAT (Stub da), Knowledge-Base (FE-Tab da, BE fehlt), Auto-Routing.

### berichte (BE berichte, hinter featureflag modules.berichte)
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| KPI-Dashboards | ✓ | ✓ | 🟢 | DashboardGrid (Recharts) |
| No-Code Query-Builder | ✗ | ◐ | 🔴 | BE liest query_config; **FE-Builder fehlt komplett** |
| Drill-Down/Drill-Through | ◐ | ◐ | 🟠 | FE-Stub; Drill-Endpoint fehlt |
| Export PDF/CSV/XLSX | ✓ | ✓ | 🟢 | useExportReport |
| Filter & Breakouts | ◐ | ◐ | 🔴 | kein Breakout/Pivot |
| Geplante Berichte + Alerts | ✓ | ✓ | 🟢 | ScheduleList; Alert-Mail unklar |
| Datenquellen-übergreifend | ◐ | ◐ | 🔴 | cross-Kind fehlt im Executor |

Positiv: DatevBridge im Executor injiziert → DATEV-Daten in Berichten (ZFA-stark). Sauber gebaut.

### team (BE route_hr.go, route_datev_upload.go)
| Feature | FE | BE | Aktion | Datei / Notiz |
|---|---|---|---|---|
| Mitgliederverwaltung | ✓ | ✓ | 🟢 | useEmployees |
| Organigramm | ✓ | ✓ | 🟢 | OrgChart (rudimentär, kein D&D-Reorg) |
| Rollen im Team | ✓ | ✓ | 🟢 | CreateEmployeeWizard, ModuleAssignmentTab |
| Digitale Personalakte | ◐ | ✓ | 🟡 | PersonnelDocuments = Mock; BE /{id}/documents da |
| Abwesenheits-Übersicht | ✓ | ✓ | 🟢 | AbsenceCalendar |
| Onboarding-Workflows | ◐ | ✗ | 🔴 | FE-TODO; Onboarding-API fehlt |
| HR-Admin-Ansicht (separat) | ✓ | ✓ | 🟢 | HRApprovalDialog; SelfServiceView teils Mock |
| DATEV-Anbindung | ◐ | ✓ | 🟡 | route_datev_upload.go = Buchungsdaten (nicht HR-Lohn); FE-Panel TODO |

Positiv: ArbZG-Compliance-Toasts. Konzeptlücke: DATEV-Upload ist Buchungs-, nicht Lohndaten.

---

## Welle 2 — System, Produktivität-Rest, Finanzen, Automatisierung, Video

### Duplikat-Klärung (Welle 2)
- **`video` vs. `meetings`:** Beide aktiv, bewusst getrennt. `meetings/` = primär (Nav-Eintrag, LiveKit, volle Verwaltung). `video/VideoPage.tsx` = Call-History-Ansicht (Dialer-Kontext), liest aus Store, Buttons **nicht funktional verdrahtet**. Kein Altlast, aber VideoPage anbinden oder zusammenführen.
- **Doppelte Profil-Impl.:** `profil/ProfilTab.tsx` + `settings/SettingsPage::ProfileTab()` — identische Felder, beide gegen lokalen Store. Konsolidieren.
- **`SecurityAdminPage` vs. `SecurityAdminHubTab`:** zwei Einstiegspunkte für identischen Content.
- **`buchhaltung` ist `_DEPRECATED.md`** — Inhalte nach `finanzen` migriert, aber noch im Routing + nutzt nur Mock-Store.
- **`settings` SecurityTab** enthält Mock-Sessions (Altlast-Stub), echte liegen in `security/SessionsPage`.

### admin
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Mandanten-/Tenant-Verwaltung | ◐ | ✓ | 🟡 | BE RLS vollständig; kein Tenant-CRUD-UI |
| Automatisches Tenant-Provisioning | ✗ | ✗ | 🟠 | Provisioning-Endpoint + Onboarding-Wizard fehlen |
| Benutzerverwaltung & Einladungen | ✓ | ✓ | 🟢 | route_auth.go /users + /invitations |
| Admin-Ebenen (Company vs. System) | ◐ | ◐ | 🟠 | Super-Admin/System-Level-Rolle fehlt |
| Lizenz-/Abo-Übersicht | ◐ | ✗ | 🔴 | Billing-Backend fehlt (statische Mock-Daten) |
| Ressourcen-Monitoring pro Tenant | ◐ | ✗ | 🔴 | Mock; Tenant-Monitoring-API fehlt |

### dashboard
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Konfigurierbare Widgets | ✓ | ✓ | 🟢 | route_dashboard.go layout |
| KPIs/Kennzahlen | ✓ | ◐ | 🟡 | aus CRM/Work-Services, kein zentraler Aggregator |
| Schnellzugriffe | ✓ | ✓ | 🟢 | QuickActionsBar |
| Aktivitäts-Feed | ✓ | ◐ | 🟡 | aus CRM, kein modul-übergreifender Feed |
| Personalisierung pro Nutzer | ✓ | ✓ | 🟢 | layout user-spezifisch |
| Modul-übergreifende Übersicht | ✓ | ◐ | 🟡 | je Widget eigener Service; MVP-ok |

### profil
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Persönliche Daten | ◐ | ✓ | 🟡 | FE lokaler Store statt PUT /users/{id} |
| Avatar/Foto | ◐ | ✗ | 🔴 | Camera-Button da, kein Upload-Endpoint |
| Spracheinstellung | ✓ | ✗ | 🔴 | FE komplett; BE-Persistenz fehlt (Multi-Device) |
| Benachrichtigungs-Präferenzen | ✓ | ✓ | 🟢 | route_notification.go preferences |
| Theme-Auswahl | ✓ | ✗ | 🔴 | FE voll; BE-Sync fehlt (Electron client-only ok) |
| Status/Presence | ◐ | ◐ | 🟡 | hardcoded "online"; usePresence-Hook unklar verdrahtet |

### security ⭐ (am vollständigsten)
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Login + passwordless/Passkeys | ◐ | ◐ | 🟠 | Login+2FA voll; Passkeys fehlen |
| MFA: TOTP/WebAuthn | ◐ | ✓ | 🟡 | TOTP-Wizard voll; WebAuthn fehlt |
| SSO: SAML/OAuth2/OIDC | ✗ | ✗ | 🟠 | fehlt (nicht pre-launch) |
| Federation (LDAP/AD) | ✗ | ✗ | 🟠 | fehlt (nicht pre-launch) |
| Session-Mgmt & Revocation | ✓ | ✓ | 🟢 | SessionsPage |
| RBAC/ABAC | ✓ | ✓ | 🟢 | RequireRole/Permission + RLS |
| Passwort-Policy | ✓ | ✓ | 🟢 | PasswordPolicyPage |
| Login-Verlauf/aktive Sessions | ✓ | ✓ | 🟢 | — |
| Unveränderbare Audit-Logs (Export) | ✓ | ✓ | 🟢 | Hash-Chain + Verify |
| DSGVO-Datenexport (Art. 30) | ✓ | ✓ | 🟢 | GDPRExportPage |
| DSAR/Betroffenenanfragen | ✓ | ✓ | 🟢 | DSARSearchPage |
| Recht auf Vergessen | ✓ | ✓ | 🟢 | GDPRErasurePage |

⚠ Fehlt: **"Passwort vergessen"-Flow** (Login-Screen + BE-Endpoint) — pre-launch wichtig.

### settings
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Workspace-Branding | ◐ | ✗ | 🔴 | nur localStorage; Persistenz-Endpoint fehlt |
| Integrationen verwalten | ◐ | ✓ | 🟡 | Bexio/DATEV/Lexware/Slack/Teams-Wizards da |
| Standard-Werte/Defaults | ✓ | ✓ | 🟢 | CompanySettingsTab |
| Modul-Aktivierung | ◐ | ✓ | 🟡 | Flag-Registry da; Admin-Toggle-UI fehlt |
| Sprach- & Regionseinstellung | ✓ | ✗ | 🔴 | FE voll; BE-Persistenz fehlt |

### notifications
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| In-App Notification-Center | ✓ | ✓ | 🟢 | NotificationCenter + Bell |
| Multi-Channel (Push/Mail/SMS) | ◐ | ◐ | 🟠 | desktop_push da; Mail/SMS im Gateway nicht exponiert |
| Pro-Nutzer Präferenzen | ✓ | ✓ | 🟢 | — |
| Do-Not-Disturb/Stummschaltung | ✓ | ✓ | 🟢 | QuietHours + Mutes |
| Gruppierung nach Modul | ✓ | ✓ | 🟢 | module_id-Filter |
| Trigger-/Workflow-basiert | ◐ | ◐ | 🟡 | Event-Integration da; Workflow-Builder-UI fehlt |

### work ⭐ (sehr vollständig)
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Aufgaben mit Zuweisung & Fälligkeit | ✓ | ✓ | 🟢 | — |
| Kanban-Board | ✓ | ✓ | 🟢 | dnd-kit, optimistisch |
| Listen-Ansicht | ✓ | ✓ | 🟢 | — |
| Gantt/Zeitleiste | ✓ | ✓ | 🟢 | nur due_date, **kein start_date** → Balken schätzend |
| Abhängigkeiten & Teilaufgaben | ✓ | ✓ | 🟢 | — |
| Kommentare & Anhänge | ✓ | ✓ | 🟢 | — |
| Automatisierte Regeln | ✓ | ✓ | 🟢 | via automation-Service |
| Projekt-Portfolios | ✗ | ✗ | 🟠 | Portfolio-Entität fehlt |
| Zeiterfassung integriert | ✓ | ✓ | 🟢 | TaskTimer + route_work_time.go |

### zeiterfassung
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Timer (Start/Stop) | ✓ | ✓ | 🟢 | via hr-hooks |
| Manuelle Zeiteinträge | ✓ | ✓ | 🟢 | Korrektur-Workflow (Manager-Genehmigung) |
| Gesetzeskonform (§3 ArbZG) | ✓ | ✓ | 🟢 | arbzg.go |
| Stundenkonten (Plus/Minus) | ◐ | ◐ | 🟠 | FE Mock-Store; Saldo-Endpoint fehlt |
| Zuordnung Kunde/Projekt/Leistung | ◐ | ◐ | 🟠 | HR-Worktime ohne customer_id/service_code |
| Pausen-/Arbeitszeit-Regeln | ✓ | ✓ | 🟢 | — |
| Auswertung & Export | ◐ | ◐ | 🟠 | Export = nur Toast; Mock-Store |

Auffällig: Modul ist Wrapper auf ZeiterfassungTab aus Profil-Modul (kein eigener Eingang).

### wiki
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Artikel/Seiten mit Editor | ◐ | ✓ | 🟡 | WikiPage nutzt Mock-Store statt useWiki.ts-Hooks |
| Versionsverlauf | ✓ | ✓ | 🟢 | — |
| Volltextsuche | ✓ | ✓ | 🟢 | FTS |
| Labels/Kategorien | ◐ | ✓ | 🟡 | FE Mock-Store |
| Verschachtelte Struktur | ✓ | ✓ | 🟢 | ParentID |
| Share-Links | ✓ | ◐ | 🟡 | Repo-Methoden da, **Route nicht registriert** |
| Inline-Anhänge/Dateien | ✓ | ✓ | 🟢 | — |

### finanzen (Vollersatz-Ziel)
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Angebote erstellen | ✓ | ✓ | 🟢 | route_biz_quotes.go |
| Rechnungen erstellen | ✓ | ✓ | 🟢 | GoBD invoice/service.go |
| E-Rechnung: XRechnung & ZUGFeRD | ◐ | ◐ | 🟠 | ZUGFeRD da; **XRechnung-UBL fehlt im BE**; FE Mock-Status |
| Angebot → Rechnung Übernahme | ✓ | ✓ | 🟢 | — |
| Wiederkehrende Rechnungen | ✗ | ✗ | 🟠 | Tabelle + Scheduler fehlen |
| Zahlungsabgleich (Bank) | ◐ | ✗ | 🟠 | BankingWidget Mock; kein Open-Banking-Endpoint |
| Fremdwährungs-Rechnungen | ✗ | ◐ | 🟠 | EUR hardcoded; currency-Feld fehlt |
| GoBD-konformes Archiv | ✓ | ✓ | 🟢 | service_gobd.go |

### buchhaltung (Brücke-Modul, _DEPRECATED → finanzen)
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| GoBD-konformes Journal | ◐ | ✓ | 🟡 | FE Mock; BE JournalSummary da |
| DATEV-Export/Schnittstelle | ✓ | ✓ | 🟢 | route_datev_upload.go (OAuth, Buchungsstapel, Beleg) |
| Belegerfassung & -archiv | ◐ | ✓ | 🟡 | FE Mock; BE Belegbilder + GoBD da |
| Automatische Kontierung | ✗ | ✗ | 🟠 | kein Kontenplan (SKR03/04) |
| Mahnwesen (mehrstufig) | ✓ | ✓ | 🟢 | DunningPanel + voll BE |
| EÜR/Auswertungen | ◐ | ◐ | 🟠 | Mock-Charts; EÜR-Endpoint fehlt |
| Steuerberater-Zugang | ✗ | ✗ | 🟠 | tax_advisor-Rolle fehlt |

### automatisierung (Brücke/„Zapier-Light", KMU-tauglich)
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Trigger/Aktion-Logik | ✓ | ✓ | 🟢 | 17 Trigger, 8 Action-Typen |
| Bedingungen (if/then) | ✓ | ✓ | 🟢 | AND/OR + expr-lang |
| Verzweigungen (branching/merge) | ◐ | ✗ | 🔴 | Engine sequenziell; Branch-Step fehlt im Modell |
| Templates | ✓ | ✓ | 🟢 | 12 Templates |
| Webhooks/API-Aufrufe | ✗ | ✗ | 🟠 | http_request-Action + inbound-Webhook-Trigger fehlen |
| Modul-übergreifende Workflows | ◐ | ✓ | 🟡 | BE deckt ab; FE-Chaining-Visualisierung fehlt |
| Self-hosted/EU-Daten | ✓ | ✓ | 🟢 | eigener Go-Service |

### video / meetings ⭐ (am ausgereiftesten)
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Video-/Audio-Call (WebRTC) | ✓ | ✓ | 🟢 | LiveKit |
| Screen Sharing | ✓ | ✓ | 🟢 | — |
| Recording | ✓ | ✓ | 🟢 | Egress-Webhook |
| Lobby/Moderation | ✓ | ✓ | 🟢 | MeetingLobby/PreJoin |
| Breakout-Räume | ✗ | ✗ | 🟠 | fehlt (LiveKit kann es technisch) |
| Consent-Banner (DSGVO) | ✓ | ✓ | 🟢 | vorbildlich (nicht wegklickbar, Snapshot in DB) |
| Browser-basiert (kein Plugin) | ✓ | ✓ | 🟢 | WebRTC |
| Self-hosted/EU-Daten | ✓ | ✓ | 🟢 | LiveKit + coturn auf Hetzner |

Fehlt: Breakout-Räume, Recording-Download-UI, Meeting-Recurrence-Logik.

## Welle 3 — Branchen-Module (Post-Launch / Solar-Pilot)

**Durchgängiger Befund:** ALLE 7 Branchen-Module folgen identisch dem Muster „BE + TanStack-Hooks + Client fertig, aber FE-Page hängt komplett an Zustand-Mock-Store (localStorage)". Kein einziger echter API-Call in den Page-Komponenten. Umstellung je Modul ~1–2 Tage FE-Arbeit. Zusätzlich modulübergreifend: Foto-Upload überall Mock (S3 fehlt), Signatur-Canvas-Persistenz fehlt im BE, Einkauf↔Inventar-Sync fehlt (Code-Kommentar „Sprint-3-Item").

### fuhrpark
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Digitale Fahrzeugakte | ◐ | ✓ | 🟡 | FE Mock-Store; useVehiclesList ungenutzt |
| Elektr. Führerscheinkontrolle | ✗ | ✗ | 🟠 | fehlt komplett |
| GPS-Ortung | ◐ | ✗ | 🟠 | FE-Tab Mock; Telematik-Provider + Webhook nötig |
| Fahrtenbuch (finanzamtkonform) | ◐ | ✗ | 🟠 | FE-Anzeige da, Add „coming soon"; BE-Modell fehlt |
| Wartung/TÜV-Reminder | ◐ | ✓ | 🟡 | BE+Worker fertig; FE Mock-Store |
| Fahrzeugbuchung (Pool) | ✗ | ✗ | 🟠 | fehlt komplett |
| Kosten-/Tankkartenverwaltung | ◐ | ✗ | 🟠 | FE-Tank-Tab Mock; FuelRecord-Modell fehlt im BE |
| Schadenmanagement | ◐ | ✓ | 🟡 | BE ReportDamage da; FE Store + Mock-Foto |

### inventar
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Artikel-Stammdaten | ◐ | ✓ | 🟡 | FE nur toast, kein API-Call |
| Bestandsführung | ◐ | ✓ | 🟡 | BewegungDialog nur toast |
| Bestands-Alarm (Mindestmenge) | ✓ | ✓ | 🟢 | maybeCreateWarning auto |
| Chargen/Seriennummern | ◐ | ✗ | 🟠 | FE-Felder da; BE-Modell fehlt |
| Inventur | ◐ | ✗ | 🟠 | FE-Tab voll (Soll/Ist); BE fehlt komplett |
| Wareneingang/-ausgang | ◐ | ◐ | 🟡 | Movement da; Einkauf→Inventar-Sync fehlt |
| Kommissionierung/Picklisten | ✗ | ✗ | 🟠 | fehlt komplett |
| Barcode/QR-Scan | ◐ | ◐ | 🟡 | FE Texteingabe (kein Kamera-Scan); Barcode-Feld im BE da |

### vermietung
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Mietobjekt-Verwaltung | ◐ | ✓ | 🟡 | FE Store; Objekt-Typen ≠ BE-category |
| Buchungen & Verfügbarkeit | ◐ | ✓ | 🟡 | CheckAvailability nicht vor Buchung verdrahtet |
| Zustandsprotokolle | ◐ | ✓ | 🟡 | FE Checklist+Signatur; BE nur notes+photo_urls |
| Preis-/Tarifmodelle | ◐ | ◐ | 🟡 | nur daily_rate+deposit; keine Staffeln/Wochensätze |
| Übergabe-Dokumentation | ◐ | ✓ | 🟡 | Signatur wird nicht ans BE übertragen |
| Online-Buchungsportal | ✗ | ✗ | 🟠 | fehlt komplett |

### einkauf
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Lieferantenverwaltung | ◐ | ✓ | 🟡 | FE Store; SupplierRating fehlt im BE |
| Bestellungen | ◐ | ✓ | 🟡 | Status-Mismatch FE(confirmed)↔BE |
| Automatische Bestellvorschläge | ✗ | ✗ | 🟠 | fehlt (Inventar-MinQty → PO) |
| Wareneingangskontrolle | ◐ | ◐ | 🟡 | BE empfängt, bucht aber nicht in Inventar |
| Preis-/Konditionsverwaltung | ◐ | ◐ | 🟡 | FrameworkContract-Tab ohne BE |
| Bestellfreigabe-Workflow | ◐ | ◐ | 🟡 | Submit da; kein 2-stufiger Approval |

### produktion (Brücke — MRP-Tiefe begrenzt)
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Produktionsaufträge | ◐ | ✓ | 🟡 | FE Mock; BE CRUD+Start/Complete da |
| Stücklisten (BOM) | ◐ | ✗ | 🔴 | FE-Tab+Dialog voll; BOM-Modell fehlt im BE |
| Maschinenbelegung/Planung | ◐ | ✓ | 🟡 | FE-Gantt Mock; Maschinen-Stammdaten fehlen (nur String-ID) |
| Kalkulation | ✗ | ✗ | 🟠 | fehlt komplett |
| Fortschritts-Tracking | ◐ | ✗ | 🔴 | progress/work_steps/scrap-Felder fehlen im BE |
| Material-Verfügbarkeit (MRP) | ◐ | ✗ | 🔴 | FE Fake-Hash; Inventar-Abgleich fehlt |

### schichten (Solar-Pilot wichtig)
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Schicht-/Dienstplan-Erstellung | ◐ | ✓ | 🟡 | FE Mock-Assignments; BE CRUD+publish da |
| Auto-Planer (regelbasiert) | ✗ | ◐ | 🔴 | ApplyTemplate ≠ echter Planer; FE-UI fehlt |
| ArbZG-Compliance-Prüfung | ◐ | ✓ | 🟡 | doppelt (FE lokal + BE-Endpoint ungenutzt) |
| Verfügbarkeiten & Qualifikationen | ◐ | ✗ | 🔴 | BE-Tabellen fehlen |
| Schichttausch durch Mitarbeiter | ◐ | ✗ | 🔴 | FE-Tab da; swap_requests-Modell fehlt |
| Mobile App-Ansicht | ✗ | — | 🟠 | Electron-Desktop; PWA/Mobile fehlt (Pilot-kritisch) |
| Minderjährigen-Regeln (Azubis) | ✗ | ✗ | 🟠 | JArbSchG fehlt (is_minor-Flag) |

### rapporte (Solar-Pilot wichtig)
| Feature | FE | BE | Aktion | Notiz |
|---|---|---|---|---|
| Mobile Rapport-Erfassung | ◐ | ✓ | 🟡 | FE Store; Desktop-only (kein Mobile-Zugang) |
| Foto-Dokumentation | ◐ | ✓ | 🟡 | FE Mock-Fotos; BE ReportAttachment da; echter Upload fehlt |
| Approval-Workflow | ✓ | ✓ | 🟢 | FE+BE modelliert (nur verbinden) |
| GPS-Tag/Standort | ◐ | ✓ | 🟡 | BE bereit (Lat/Lon); FE kein geolocation-Call |
| Material & Leistung erfassen | ◐ | ◐ | 🟡 | BE ReportLine generisch; Typ-Mapping nötig |
| Offline-Fähigkeit | ◐ | — | 🟡 | offline-queue.ts da; rapporte-client nutzt ihn nicht |
| Digitale Unterschrift vor Ort | ◐ | ✗ | 🔴 | SignatureCanvas da; BE-Persistenz fehlt |

Auffällig rapporte: Aufmaß-Tab (Measurement) komplett im FE, **null** BE — „FE läuft vor BE".

---

# GESAMT-SCORECARD (32 Module, 235 Features)

**Tendenz:** Solide Backend-Basis fast überall. Hauptarbeit ist **Frontend an fertige API anklemmen** (🟡), nicht Neubau. Echte Backend-Lücken (🔴) konzentrieren sich auf: Online-Terminbuchung, E-Invoice/Banking, MRP/BOM, Schicht-Self-Service, Signatur-Persistenz, S3-Foto-Upload.

| Reife-Stufe | Module |
|---|---|
| ⭐ Nahezu fertig (BE+FE) | security, video/meetings, work, notifications, automatisierung (Zapier-Light), dialer (Kern) |
| 🟡 Gut, v.a. FE-Anbindung nötig | crm, kontakte, kalender, dokumente, formulare, wiki, berichte, team, helpdesk, vertraege, finanzen |
| 🟡🔴 FE-Anbindung + spürbare BE-Lücken | mails, buchhaltung, fuhrpark, inventar, vermietung, einkauf |
| 🔴 Substanzielle BE-Lücken | produktion (BOM/MRP), schichten (Self-Service/Planer), rapporte (Signatur/Mobile) |
| 🗑 Altlast/Konsolidieren | calendar (löschen), chat→kommunikation (mergen), profil↔settings-Doppel, buchhaltung _DEPRECATED |

**Cross-cutting (modulübergreifend, einmal lösen → viele Module profitieren):**
1. S3/MinIO-Foto-Upload (fuhrpark, inventar, rapporte, vermietung, chat, profil-Avatar)
2. Signatur-Persistenz-Pfad (rapporte, vermietung, vertraege)
3. Mobile/PWA + offline-queue-Anbindung (rapporte, schichten — Solar-Pilot)
4. Einheitliches „FE-Page von Mock-Store auf TanStack-Hooks umstellen" (fast jedes Modul)
5. DSGVO-Consent-Konstruktor-Fix (dialer) + Passwort-vergessen-Flow (security)
