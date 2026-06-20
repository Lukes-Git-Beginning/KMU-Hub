# Formulare → Verteilung & Lebenszyklus — VISION & Plan (Handoff)

> **Stand 2026-06-20.** Pivot nach Darien-Feedback: Das formulare-Modul baut Formulare, aber die gesamte Schicht danach fehlt — Veröffentlichen, Live-Schalten, Teilen, Ausfüllen durch Externe, Antworten-Auswertung, Formular schließen. Diese Vision definiert die fehlende „Verteilungs- und Lebenszyklus-Schicht" (FD-0…FD-5) sowie die gemeinsame öffentliche Web-Schicht mit dem berichte-Modul.
>
> **Der bisherige Builder (F-1…F-5, gemergt 2026-06-20) bleibt unverändert** — er liefert das `FormSchema` als Eingabe für alles hier.

---

## 1. Darien-Vision (O-Ton, vollständig)

1. Formulare „liegen nur rum" nach dem Erstellen — der gesamte Bereich **wo das Formular angezeigt / ausgefüllt wird** und **wo Antworten zusammenlaufen**, fehlt.
2. Beim Teilen fehlt eine **Übersicht**: wie lange ist der Link gültig, wo / an wen wurde geteilt (Verteil-Historie, Aufrufe, Conversion).
3. Offene Frage: Läuft das Ausfüllen über einen **echten Website-Link**, wenn jemand OHNE Cosmi das Formular ausfüllen will?

---

## 2. Gap-Analyse (verifiziert im Code, Stand 2026-06-20)

### 2a. Was bereits vorhanden ist

| Bereich | Stand |
|---|---|
| Formular-Builder (mehrseitig, DnD, Conditional-Logic, DSGVO-Consent-Feld) | vollständig (F-1…F-5) |
| `isPublic`-Toggle pro Schema | vorhanden (MSW + Backend-Modell) |
| In-App-Vorschau (Modal mit „Vorschau"-Banner, State `showPublicPreview`) | vorhanden, aber NICHT extern erreichbar |
| Submissions-Sammlung (`POST /api/v1/formulare/schemas/:id/submissions`) | MSW + Backend-Route |
| Submissions-Liste + Statuswechsel (new / read / archived) | vorhanden |
| CSV- und XLSX-Export (`GET /schemas/:id/submissions/export`) | vorhanden, echter Blob-Download |
| Stats-Endpoint (`GET /schemas/:id/stats`) | vorhanden |
| Webhooks (CRUD + Deliveries-Observability) | vorhanden |
| Schema-Status-Modell: `draft` / `active` / `archived` | Typen vorhanden, aber kein Lebenszyklus-Guard |

### 2b. Was fehlt — fünf Lücken im Detail

**(a) Öffentliche Ausfüll-Seite für Externe**

Aktuell ist **kein einziger Backend-Endpoint ohne `RequireAuthenticated`-Middleware** erreichbar. Die Submission-Route `POST /schemas/:id/submissions` verlangt ein gültiges JWT und `tenant_id`. Externe ohne Cosmi-Account können kein Formular sehen oder absenden. Es gibt keinen Web-Client (Cosmi = Electron), keine öffentliche HTML-Seite, keine Token-Route.

**(b) Share-Link mit Gültigkeit / Ablauf / Passwort / Antwort-Limit / Schließdatum**

Der Teilen-Dialog in `FormularePage.tsx` (Zeile 2721–2802) ist ein reiner Dummy:
- „Link kopieren" → `toast.success(t('formulare.toast.linkKopiert'))` — keine URL wird gebildet, keine Zwischenablage-Beschreibung, kein Token.
- „Per E-Mail" → `toast.success(t('formulare.toast.emailGesendet'))` — kein Mailer, kein API-Call.
- Kein `shareToken`-Feld im `FormSchema`-Typ (`formulare-types.ts`).
- Keine Ablaufdatum-, Passwort-, Antwort-Limit-, Schließdatum-Felder weder im Typ noch im MSW-Handler.

**(c) Verteil-Historie „wo / wann / an wen geteilt" + Aufrufe / Conversion**

Vollständig fehlend. Es gibt keine `form_share_links`-Tabelle, kein `shared_at`-Feld, keine Kanal-Kennzeichnung (E-Mail / QR / Link-Kopie / Einbettung), keinen View-Counter, keine Conversion-Rate (Aufrufe → Submissions).

**(d) Antworten-Auswertung im Modul**

Stats-Endpoint liefert Zählungen (total / new / read / archived / thisWeek / thisMonth / averageCompletionRate). Felder-übergreifende Auswertung (welche Antwort-Option wurde wie oft gewählt, Trichteranalyse, Drop-off pro Seite bei mehrseitigen Formularen) fehlt. Kein Diagramm, keine Pivot-Ansicht.

**(e) Formular-Lebenszyklus**

Das Typ-Modell kennt `draft | active | archived`. In der UI gibt es Status-Badges, aber keine Guards: Ein `draft`-Formular kann per `isPublic`-Toggle trotzdem geteilt werden. Es gibt keinen „Veröffentlichen"-Button, kein explizites „Live-Schalten", kein Schließen/Pausieren mit Zeitstempel, keine Closed-Message-Konfiguration im Frontend, kein Status-Übergangs-Protokoll.

---

## 3. Markt-Recherche-Synthese (Tally / Typeform / JotForm / Fillout / Microsoft Forms)

### 3a. Lebenszyklus-Standard im Markt

Alle relevanten Anbieter trennen klar zwischen **Erstellen** und **Veröffentlichen/Live-Schalten**. Ein neu erstelltes Formular ist immer im `Entwurf`-Zustand und nicht öffentlich. Erst ein expliziter „Publish"- oder „Go Live"-Schritt schaltet die öffentliche URL scharf.

| Status | Tally | Typeform | JotForm | Microsoft Forms | Fillout |
|---|---|---|---|---|---|
| Entwurf (nicht öffentlich) | ja | ja | ja | ja | ja |
| Live / Veröffentlicht | ja, manuell | ja, manuell | ja, manuell | ja, sofort | ja, über „Publish"-Button |
| Pausiert / Geschlossen | ja (manuell + Datum) | ja (Datum + Limit) | ja (Datum + Limit) | eingeschränkt | ja |
| Archiviert | nein | nein | ja | nein | nein |

### 3b. Share-Link-Optionen

| Feature | Tally | Typeform (Business) | JotForm | Fillout | MS Forms |
|---|---|---|---|---|---|
| Antwort-Limit | ja | ja | ja | unklar | nein |
| Schließdatum | ja | ja | ja | unklar | nein |
| Passwortschutz | ja | nein | ja | nein | nein |
| Mehrfach-Submission-Schutz | ja (Feld als Unique-Key) | begrenzt | ja | nein | eingeschränkt |
| Closed-Message (angepasst) | ja | ja | ja | ja | begrenzt |
| Nach-Submit-Redirect | ja | ja | ja | ja | ja |
| Danke-Seite (anpassbar) | ja | ja | ja | ja | ja |
| Einbettung (iFrame / Popup / Fullscreen) | ja | ja | ja | ja (4 Varianten) | ja |
| QR-Code (nativ) | nein (Drittanbieter) | ja | ja | unklar | ja |
| E-Mail-Versand (im Tool) | nein (nur Respondenten-CC) | ja (Typ) | ja | ja | ja (Outlook/Teams) |

**Ablaufdatum für den Link selbst** (nicht für das Formular als ganzes) ist bei keinem der geprüften Anbieter Standard. Ablaufdatum = Formular-Schließdatum. Das ist die richtige KMU-Vereinfachung.

### 3c. Verteil-Übersicht / Kanal-Tracking

Typeform und JotForm zeigen in der Auswertung einen „Referral"-Aufschlüsselung: woher kamen die Submissions (direkter Link, eingebettet, E-Mail). Tally zeigt keine Kanal-Aufteilung. Für Cosmi-KMU: **eine Kanalliste pro Formular** (wann, wohin geteilt, wie viele Aufrufe und Submissions darüber) ist eine echte Alleinstellung gegenüber einfacheren Tools.

### 3d. Öffentliche Ausfüll-Seite

Alle genannten Anbieter hosten eine **öffentlich erreichbare HTML-Seite** unter einer eigenen Domain (z.B. `tally.so/r/xyz`, `form.typeform.com/to/xyz`). Cosmi hat keine solche Web-Komponente. Das ist der kritischste fehlende Baustein.

### 3e. DSGVO-Relevanz für DACH-KMU (Prioritäten)

**Unbedingt nötig (P0):** Consent-Feld bereits vorhanden; Datenlokation EU-only (Hetzner = bereits gewährleistet); Zweckbindung auf der Ausfüll-Seite sichtbar; Submissions-Löschung möglich.

**Sinnvoll (P1):** Ablaufdatum für Formulare, Antwort-Limit, IP-Adressen-Protokollierung (bereits im `FormSubmission`-Typ vorhanden).

**Overkill für KMU-MVP (weglassen):** Respondenten-Authentifizierung via SSO, HIPAA-Alignment, vollständiges Audit-Log jedes Feld-Zugriffs, Double-Opt-in im Formular-Tool (gehört in CRM/Newsletter).

---

## 4. Architektur — gemeinsame öffentliche Web-Schicht

### 4a. Ist-Situation

Cosmi = Electron-Desktop-App (kein Browser-Client für Externe). Das Go-Gateway hat **ausschließlich auth-gesicherte Routen** (alle unter `r.Use(authMiddleware)` / `RequireAuthenticated`). Eine externe Person ohne Cosmi-Login kann weder ein Formular sehen noch absenden. Gleiches Problem beim berichte-Modul (externer Token-Read-Link, ebenfalls noch nicht existent).

### 4b. Drei Optionen für öffentliche Token-Auslieferung

---

**Option A — Öffentliche Gateway-Routen + schlankes Server-Side HTML (Empfehlung)**

Der Go-Gateway bekommt einen neuen, auth-freien Routeblock `/public/`:

```
GET  /public/forms/{token}           → HTML-Seite (schlankes Go-Template oder Prebuilt React-Bundle)
POST /public/forms/{token}/submit    → Submission anlegen (rate-limited, kein JWT)
GET  /public/reports/{token}         → HTML-Leseansicht Bericht (berichte-Modul, gleiche Schicht)
```

Token = kryptografisch zufälliger Wert (`crypto/rand`, 32 Byte, URL-safe Base64), gespeichert in Tabelle `form_share_links` mit `schema_id`, `token`, `expires_at`, `password_hash`, `max_submissions`, `created_by`, `created_at`. Beim Aufruf: Token validieren → Schema laden → HTML rendern → CORS-Header für Einbettung.

Die Ausfüll-Seite ist ein minimales, vom Gateway ausgeliefertes HTML (einfaches Go-Template mit inline CSS oder kleines separates React-Bundle unter `web/public-form/`) — kein Electron, kein volles Cosmi-Bundle.

**Vorteile:** Kein separater Hosting-Aufwand, gleiche Infra (Hetzner-Gateway), Rate-Limiting im Gateway, DSGVO-Lokation gewährleistet, gemeinsam mit berichte-Token-Schicht nutzbar.

**Nachteil:** Luke-Aufwand für Go-Template / schlankes Web-Bundle + neue Routen. Ist aber der sauberste Ansatz.

---

**Option B — Separater schlanker Next.js / Astro-Service auf eigener Subdomain**

Zweite Web-App unter z.B. `forms.zentria.tech`, deployed auf Hetzner oder Vercel EU. Ruft intern das Gateway-API auf (`GET /api/v1/public/forms/{token}` mit Service-Account-Token) und rendert serverseitig.

**Vorteile:** Vollständig vom Electron-App-Bundle trennbar, theoretisch mit eigenem CDN.

**Nachteil:** Zusätzlicher Service zu betreiben, zusätzliche Deployment-Pipeline, mehr Komplexität für das 1-Dev-Team. Außerdem: Vercel ist US-Firma — DSGVO-konformes EU-Deployment erfordert Vercel EU-Region oder Selbsthosting.

---

**Option C — Einbettungs-Only (iFrame) ohne eigene öffentliche URL**

Kein öffentlicher Link. Cosmi generiert nur einen iFrame-Code-Snippet (`<iframe src="https://app.zentria.tech/embed/forms/{token}">`) — der Kunde bettet das Formular auf seiner eigenen Website ein. Submissions gehen an den Gateway.

**Vorteile:** Kein neuer Web-Service, keine eigenständige Seite zu pflegen.

**Nachteil:** Externe ohne Website (oder mit statischer Seite) haben keinen einfachen Zugang. QR-Code-Sharing (auf Flyern, Events) geht nicht. Deutlich eingeschränkter als Markt-Standard.

---

### 4c. Empfehlung: Option A

Option A ist der richtige Ansatz für Cosmi. Das Gateway ist bereits in Go, die Infra ist auf Hetzner, der Zusatzaufwand ist überschaubar (1 neue Route-Gruppe, 1 Tabelle, 1 schlankes HTML-Template oder kleines React-Bundle). Option B lohnt sich erst ab ~Phase E (PWA), wenn sowieso ein zweiter Web-Client entsteht. Option C ist für den Marktvergleich nicht wettbewerbsfähig.

**Gemeinsamer Backend-Baustein mit berichte:** Die `public_tokens`-Tabelle und die `/public/`-Route-Gruppe können von beiden Modulen genutzt werden — `resource_type: 'form' | 'report'`, `resource_id: UUID`. Das ist der richtige Zeitpunkt, diesen Baustein einmal sauber zu bauen (nicht zweimal).

---

## 5. Infrastruktur-Ist (was da ist / was fehlt)

### Bereits vorhanden (wiederverwenden)

| Baustein | Status |
|---|---|
| `FormSchema`-CRUD (Builder F-1…F-5) | vollständig |
| `FormSubmission`-CRUD (MSW + Backend-Handlers) | vollständig |
| Submission-Status-Wechsel (new / read / archived) | vollständig |
| CSV/XLSX-Export | vollständig |
| Stats-Endpoint | vollständig |
| Webhooks-CRUD + Deliveries | vollständig |
| DSGVO-Consent-Feld im Builder | vollständig |
| IP-Adresse in `FormSubmission` | im Typ vorhanden, Backend füllt |
| `isPublic`-Toggle | vorhanden, aber unverbunden |
| `shared/DetailModal` | wiederverwendbar für Submission-Detail |
| `shared/SortMenu` | wiederverwendbar in Submissions-Tab |
| `shared/RichTextEditor` | für zukünftige Danke-Seite / Closed-Message |

### Fehlt (zu bauen)

| Lücke | FE-mockbar (MSW)? | Echter Backend-Bedarf (Luke)? |
|---|---|---|
| `FormShareLink`-Entität (Token, Ablauf, Passwort, Limit, Kanäle) | ja, als MSW-State | ja (Tabelle, Token-Generierung, Validierung) |
| Öffentliche Gateway-Route `/public/forms/{token}` | nein | ja — kritischer P0 |
| Öffentliche Ausfüll-Seite (HTML/React-Bundle) | nein | ja (Go-Template oder Web-Bundle) |
| Rate-Limiting für `/public/submit` | nein | ja |
| Share-Dialog mit echter URL-Generierung | ja (MSW generiert Dummy-Token) | ja (echter Token vom Backend) |
| Verteil-Kanäle (E-Mail, Einbettung, QR, Link) | ja (MSW) | ja (Mailer, Delivery-Log) |
| Lebenszyklus-Guards (Entwurf darf nicht geteilt werden) | ja (MSW) | ja |
| Antwort-Limit + Schließdatum (Formular schließt automatisch) | ja (MSW simuliert) | ja (Backend-Check beim Submit) |
| Closed-Message konfigurierbar | ja (MSW) | ja |
| Verteil-Historie + Aufrufe / Conversion | ja (MSW-Seed) | ja (View-Counter, Delivery-Log) |
| Feld-Auswertung (Antwort-Verteilung pro Feld) | ja (MSW berechnet aus Submissions) | ja |
| QR-Code-Generierung | ja (Dummy-PNG-Platzhalter) | ja (Server-seitige QR-Generierung oder clientseitig via `qrcode` npm) |
| Nach-Submit-Weiterleitung / Danke-Seite | ja (MSW) | ja (auf öffentlicher Seite) |

---

## 6. Phasen-Plan FD-0…FD-5

| # | Phase | Inhalt | FE-mockbar? | Luke-Bedarf? |
|---|---|---|---|---|
| **FD-0** | Lebenszyklus-Guards + Status-Modell | Status `Entwurf → Live → Geschlossen → Archiviert` scharf verdrahten. „Veröffentlichen"-Button (Entwurf→Live). „Formular schließen" mit Closed-Message (eigenes Textfeld). Guard: Teilen-Button nur für Live-Formulare aktiv. Status-Badge im Formular-Header aktualisieren. MSW-Lebenszyklus-Übergänge. | ja | nein (MSW-only bis FD-4) |
| **FD-1** | Share-Dialog (echter Link + Kanal-Erfassung) | Share-Dialog komplett ersetzen: URL-Anzeige mit Dummy-Token (z.B. `https://forms.zentria.tech/r/{token}`), Kopieren-Button (echter `navigator.clipboard.writeText`), Kanal wählen (Link / E-Mail / Einbettung / QR), Ablaufdatum setzen, max. Antworten setzen, optionaler Passwortschutz. MSW: neuer State `SHARE_LINKS`, Endpoint `POST /schemas/:id/share-links`, `GET /schemas/:id/share-links`. I18n-Schlüssel `formulare.share.*`. | ja | nein (Token-Generierung in MSW) |
| **FD-2** | Verteil-Übersicht + Kanal-Historie | Neuer Sub-Tab „Teilen" oder Panel in der Formular-Detailansicht: Liste aller aktiven und abgelaufenen Share-Links (Kanal-Icon, erstellt am, abläuft am, Aufrufe, Submissions über diesen Link, Status aktiv/abgelaufen). QR-Code-Vorschau und Download. „Einbettungs-Code" kopieren. Archivieren / Löschen einzelner Links. MSW: View-Counter wird bei jedem `GET`-Aufruf des Tokens erhöht (simuliert). | ja | nein |
| **FD-3** | Antworten-Auswertung | Neuer Auswertungs-Tab in der Formular-Detail-Ansicht: Feld-übergreifende Analyse (Balkendiagramm für select/radio/checkbox, Freitext-Vorschau-Liste für text/textarea, Datum-Verteilung für date-Felder). Drop-off pro Seite bei mehrseitigen Formularen (% haben Seite X abgebrochen). Conversion-Rate (Aufrufe → Submissions, nur wenn Token-Aufrufe bekannt). Wiederverwendung: `ChartRenderer`-Komponente aus berichte (Schicht 3). | ja | nein |
| **FD-4** | Öffentliche Ausfüll-Seite (echter Backend-Bedarf) | Neuer öffentlicher Gateway-Routeblock `/public/forms/{token}`. Schlanke Ausfüll-Seite (HTML-Template oder kleines React-Bundle `web/public-form/`): rendert Formular-Felder, DSGVO-Consent klar sichtbar, mehrseitig mit Fortschrittsbalken, Nach-Submit-Danke-Seite / Redirect. Kein Cosmi-Login erforderlich. Rate-Limiting (z.B. 10 Submits/min/IP). Token-Validierung (Ablauf, Passwort, Antwort-Limit). Gemeinsame Tabelle `public_tokens` mit berichte-Modul. | nein | **ja — Luke, P0** |
| **FD-5** | Weitere Verteilungs-Kanäle | E-Mail-Versand (Empfänger eingeben, personalisierter Link), E-Mail-Template mit Formular-Vorschau. QR-Code server-seitig generiert (via `github.com/skip2/go-qr` oder clientseitig via `qrcode`-npm-Paket). Einbettungs-Code-Snippet mit 4 Varianten (Standard-Block / Popup / Fullscreen / Slider) analog zu Fillout. Ggf. benutzerdefinierter Slug (`/r/kundenfeedback` statt `/r/a1b2c3`). | teilweise (Snippet-Anzeige) | **ja — Luke (Mailer, QR-Server, Slug-Unique-Check)** |

**Reihenfolge:** FD-0 und FD-1 sind rein FE-seitig (MSW) und können sofort ohne Luke starten. FD-2 und FD-3 ebenfalls. FD-4 ist der einzige echte Blocker (echter öffentlicher Endpoint). FD-5 hängt von FD-4 ab.

---

## 7. UI-Bausteine (konkrete Beschreibungen)

### Lebenszyklus-Status-Badges

Vier Status, farblich klar unterscheidbar (kein Emoji, custom CSS-Klassen):

| Status | Farbe | Bedeutung |
|---|---|---|
| Entwurf | `bg-secondary text-muted-foreground` | Nicht öffentlich, noch in Bearbeitung |
| Live | `bg-success-light text-success` | Aktiv, öffentlich zugänglich |
| Geschlossen | `bg-warning-light text-warning` | Nimmt keine neuen Antworten mehr an |
| Archiviert | `bg-secondary text-muted-foreground` (matter) | Inaktiv, schreibgeschützt |

Übergänge: Entwurf → Live (expliziter „Veröffentlichen"-Button, Guard: mindestens 1 Feld + Consent), Live → Geschlossen (Schließen-Aktion oder automatisch via Limit/Datum), Live → Archiviert (direkt möglich), Geschlossen → Archiviert, Archiviert → Entwurf (als Neustart). Kein Rückweg Geschlossen → Live ohne explizites Wiedereröffnen (verhindert versehentliches Reaktivieren abgelaufener Aktionen).

### Share-Link-Dialog (FD-1, Ersatz des aktuellen Dummys)

Zweistufig:
1. **Stufe 1 — URL anzeigen:** Formular-Name, generierter Link (readonly Input + „Kopieren"-Button mit echtem `navigator.clipboard.writeText`), Kanal-Auswahl-Buttons (Link / E-Mail / Einbettung / QR).
2. **Stufe 2 — Einstellungen:** Ablaufdatum (Datepicker, optional), Max. Antworten (Zahl-Input, optional), Passwortschutz (Toggle + Passwort-Input, optional). Speichern = `POST /schemas/:id/share-links`.

Link-Format in der Anzeige: `https://forms.zentria.tech/r/{token}` (Platzhalter bis FD-4 real).

### Verteil-Übersicht (FD-2)

Tabelle in einem Sub-Tab „Teilen" innerhalb der Formular-Detail-Ansicht (`DetailModal`):

| Spalte | Inhalt |
|---|---|
| Kanal | Icon + Label (Link, E-Mail, QR, Einbettung) |
| Erstellt am | Datum |
| Abläuft am | Datum oder „Kein Ablauf" |
| Aufrufe | Zähler |
| Antworten | Zähler (mit Conversion-Rate in Klammern) |
| Status | Badge (aktiv / abgelaufen / manuell deaktiviert) |
| Aktionen | Link kopieren, QR herunterladen, deaktivieren, löschen |

### Antworten-Auswertung (FD-3)

Unter Tab „Auswertung" in der Formular-Detail-Ansicht:
- Kennzahlen-Zeile: Gesamt / Neue / Conversion-Rate / Ø Bearbeitungszeit
- Pro Feld: Balkendiagramm (select/radio/checkbox), Häufigkeitsliste (text/textarea, Vorschau 5 häufigste Antworten), Kalender-Heatmap (date), Verteilungshistogramm (number).
- Seitenweise Drop-off bei mehrseitigen Formularen (Trichter-Diagramm: Seite 1 — 100%, Seite 2 — 74%, Seite 3 — 61%).

---

## 8. Offene Entscheidungen für Darien

1. **Eigene Domain für öffentliche Formulare?** `forms.zentria.tech` vs. `app.zentria.tech/forms/r/{token}`. Eigene Subdomain ist professioneller und von der App trennbar. Entscheidung beeinflusst Option A vs. B.
2. **Passwortschutz: wirklich für KMU-MVP nötig?** Tally und Typeform bieten es, aber es ist selten genutzt. Vereinfachung: erst in FD-5 oder weglassen.
3. **QR-Code: clientseitig (npm `qrcode`, sofort mockbar) oder serverseitig (Luke)?** Clientseitig ist für den MVP ausreichend und spart Luke-Aufwand — empfohlen.
4. **Antworten-Auswertung (FD-3) oder öffentliche Seite (FD-4) zuerst?** FD-4 ist der stärkere Kundenmehrwert (externe Ausfüller = Haupt-Usecase). FD-3 ist rein FE-seitig und kann parallel gebaut werden.
5. **Geschlossen-Status auf öffentlicher Seite:** Zeigt Cosmi eine generische Meldung oder eine vom Betreiber konfigurierbare Nachricht? Konfigurierbare Closed-Message ist Marktstandard und sollte in FD-0 mit rein (Textfeld im Formular-Editor).

---

## 9. Backend-Bedarf für Luke (zusammengefasst)

Alles in FD-0…FD-3 ist rein FE-seitig (MSW) und braucht Luke **nicht**. FD-4 und FD-5 sind echter Backend-Bedarf.

| Aufgabe | Phase | Priorität |
|---|---|---|
| Tabelle `public_tokens` (`resource_type`, `resource_id`, `token`, `expires_at`, `password_hash`, `max_submissions`, `view_count`, `created_by`) | FD-4 | P0 |
| Öffentliche Gateway-Route-Gruppe `/public/` ohne `RequireAuthenticated` | FD-4 | P0 |
| `GET /public/forms/{token}` — Token validieren, Schema laden, HTML ausliefern | FD-4 | P0 |
| `POST /public/forms/{token}/submit` — Submission anlegen, keine JWT-Pflicht, Rate-Limit, Antwort-Limit prüfen | FD-4 | P0 |
| `GET /public/reports/{token}` — analog für berichte (gemeinsame Schicht) | FD-4/R-5 | P0 (gemeinsam planen) |
| `POST /api/v1/formulare/schemas/:id/share-links` — Token generieren, in `public_tokens` speichern | FD-4 | P1 |
| `GET /api/v1/formulare/schemas/:id/share-links` — Liste der Tokens pro Schema | FD-4 | P1 |
| Mailer: E-Mail-Versand mit personalisierten Link-URLs | FD-5 | P2 |
| QR-Code serverseitig (nur falls clientseitig abgelehnt) | FD-5 | P3 |
| Slug-System für eigene Link-Bezeichner | FD-5 | P3 |

**Wichtig:** Die `public_tokens`-Tabelle sollte **einmal gemeinsam** für formulare und berichte designed werden (nicht zweimal separat). Koordination Luke/Darien/Claude bevor FD-4 startet.

---

## 10. Verweise

- Bisheriger Formular-Builder (F-1…F-5): gemergt in `main`, `eb379a22`
- MSW-Handler: `desktop/src/renderer/src/mocks/handlers/formulare.ts`
- Typ-Modell: `desktop/src/renderer/src/api/formulare-types.ts`
- Backend-Routen (alle auth-gesichert): `backend/internal/gateway/route_formulare.go`
- Paralleles berichte-Vision-Dokument (Token-Schicht R-5): `.planning/berichte-report-authoring-VISION.md`
- Wiederverwenden: `components/shared/DetailModal`, `shared/SortMenu`, `ChartRenderer` (berichte E-1), `shared/RichTextEditor` (für Closed-Message-Feld)
- Modul-Tracker: `.planning/MASTER-TRACKER.md`
