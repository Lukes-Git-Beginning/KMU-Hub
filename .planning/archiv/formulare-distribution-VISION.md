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

### Bestehende Entscheidungen (aus FD-Plan)

1. **Eigene Domain für öffentliche Formulare?** `forms.zentria.tech` vs. `app.zentria.tech/forms/r/{token}`. Eigene Subdomain ist professioneller und von der App trennbar. Entscheidung beeinflusst Option A vs. B.
2. **Passwortschutz: wirklich für KMU-MVP nötig?** Tally und Typeform bieten es, aber es ist selten genutzt. Vereinfachung: erst in FD-5 oder weglassen.
3. **QR-Code: clientseitig (npm `qrcode`, sofort mockbar) oder serverseitig (Luke)?** Clientseitig ist für den MVP ausreichend und spart Luke-Aufwand — empfohlen.
4. **Antworten-Auswertung (FD-3) oder öffentliche Seite (FD-4) zuerst?** FD-4 ist der stärkere Kundenmehrwert (externe Ausfüller = Haupt-Usecase). FD-3 ist rein FE-seitig und kann parallel gebaut werden.
5. **Geschlossen-Status auf öffentlicher Seite:** Zeigt Cosmi eine generische Meldung oder eine vom Betreiber konfigurierbare Nachricht? Konfigurierbare Closed-Message ist Marktstandard und sollte in FD-0 mit rein (Textfeld im Formular-Editor).

### Neue Entscheidungen (aus Modul-Tiefe-Analyse)

6. **Submission-Filterung: welche Filter-Dimensionen?** Aktuell: nur Status-Filter im MSW-Backend vorhanden. Für FT-1 muss entschieden werden: Nur Status-Filter (MVP) oder zusätzlich Datum-Range-Filter und Feld-Wert-Filter (vollständig)? Letztes erfordert eigene Filter-Komponente — nicht `shared/SortMenu` allein.
7. **`closed`-Status als eigener Typ oder Kombination?** Das Typ-Modell kennt nur `draft | active | archived`. Ein vierter Status `closed` (temporär pausiert, Submissions blockiert, reaktivierbar) passt semantisch zwischen `active` und `archived`. Typ-Erweiterung + Backend-Migration erforderlich. Alternative: `archived` für beide Zwecke nutzen (semantisch unpräzise, aber einfacher). Muss vor FT-1 entschieden sein.
8. **Submission-Detailmodal: Antworten editierbar?** Kein Marktstandard erlaubt die Bearbeitung von Submissions nachträglich. Cosmi sollte es ebenfalls nicht tun (DSGVO-Audit-Trail). Aber: Soll es eine interne Notiz-/Kommentar-Funktion am Submission-Detail geben (analog zu CRM-Notizen)? Diese würde echten Backend-Bedarf erzeugen, wäre aber ein echter Alleinstellungsmerkmal gegenüber Tally/Google Forms.
9. **Builder: Validierungs-Regeln — Scope?** Aktuell gibt es nur `required: true/false`. Für FT-2 muss der Umfang festgelegt werden: Nur Mindest-/Höchstlänge (text/number) als MVP? Oder auch Regex-Pattern (IBAN, Telefon, PLZ für DACH)? Pattern-Validierung ist DACH-KMU-relevant (z.B. PLZ-Feld), kostet aber UI-Aufwand.
10. **Submission-Tabelle in Eingänge-Tab: Sortierung per SortMenu?** Aktuell ist die Tabelle fix absteigend nach `submittedAt` sortiert (MSW sortiert so). Ein `shared/SortMenu` (Feld + Richtung) wäre projektweiter Standard — aber der Eingänge-Tab hat eine ungewöhnliche Group-by-Schema-Struktur (Akkordeon), in der SortMenu pro Gruppe angewendet werden müsste, nicht global. Entscheidung: Sortierung pro Gruppe oder globale Tabelle ohne Gruppierung?

---

---

## 9. Tiefe-Analyse: Ist-Zustand vs. Projektweit-Standard

> **Zweck dieses Abschnitts:** Vollständige Bestandsaufnahme aller Modul-Tiefe-Lücken (unabhängig von der Distributions-Schicht). Grundlage für den Phasenplan FT-1…FT-5 in Abschnitt 10.

### 9a. Funktionale Tiefe

| Bereich | Ist | Soll (projektweiter Standard) | Lücke |
|---|---|---|---|
| Submission-Detailansicht | `DetailModal` geöffnet per Zeilen-Klick, zeigt Absender, IP, DSGVO-Consent-Block, alle Antworten feldtyp-spezifisch gerendert (Checkbox, Select, Datum, Datei-Badge) | Vollständig | Keine Lücke — **hier ist das Modul bereits Referenz** |
| Formular-Detailansicht | `DetailModal` beim Karten-Klick: Metadaten, Feldliste, alle Aktionen im Footer | Alle Infos + Funktionen im Modal | Lücke: **kein Tab für Submissions oder Auswertung innerhalb des Formular-Detail-Modals** — Submissions sind nur global im Eingänge-Tab erreichbar, nicht kontextgebunden am Formular |
| Export | Echter Blob-Download (CSV/XLSX) via `GET /schemas/:id/submissions/export` | Echter Download, kein Toast-Stub | Vollständig. Einziger echter Download im Modul — korrekt |
| Share-Dialog | Reiner Dummy (zwei Toast-Stubs, kein URL, kein Clipboard, kein API-Call) | Echter Link, echter Clipboard-Write | Massiv lückenhaft — behandelt in FD-1 |
| Vorlagen-Tab | Vorhanden, Einzel-Vorlage zeigt Felder, „Verwenden"-Button funktioniert | Review-reif | Karte ist nicht klickbar (kein Detail-Modal), Vorlage nur schwer bewertbar |

### 9b. UX-Vollständigkeit

| UX-Muster | Ist | Soll | Lücke |
|---|---|---|---|
| EmptyState mit Aktion | Vorhanden: alle drei Tabs + Submission-Leer-Zustand + Builder-Leer-Zustand + Fehler-Zustand | Vorhanden + Aktions-Button | Vollständig |
| Skeleton-Loading | Aktuell: `animate-pulse`-DIVs im Skeleton (Formulare-Tab) + SubmissionsPanel-Skeleton (3 Divs) | Shimmer-Skeleton statt Spinner | Shimmer-Divs vorhanden, aber das Loading auf dem Eingänge-Tab zeigt **keinen Skeleton pro Gruppe** — nur das Akkordeon lädt einzeln. Kein globaler Submissions-Lade-Zustand |
| Sticky Zurück-/Close-Buttons | `DetailModal` sticky-Header vorhanden (kommt von `shared/DetailModal`). Editor-Zurück-Button ist im fixen Header | Immer sichtbar | Vollständig — `shared/DetailModal` löst das strukturell |
| Sortierung (SortMenu) | Keine Sortierung im Formulare-Tab (Grid, keine Table-Spalten). Submissions-Tabelle fix nach Datum absteigend | `shared/SortMenu` mit Feld + Richtung | Lücke: **Formulare-Tab: kein Sortier-Menü** (nur Text-Suche). Submissions-Tabelle: kein Sortier-Menü |
| Ansichts-/Dichte-Optionen | Keine — Formulare-Tab ist fest als Karten-Grid gebaut | Grid/List-Toggle oder Dichte-Wechsel sinnvoll | Lücke: **kein View-Toggle** (Grid vs. Liste). Bei vielen Formularen wäre Listenansicht effizienter. Kein Dichte-Toggle |
| Filter (über Suche hinaus) | Nur Text-Suche. Kein Status-Filter, kein Datum-Filter im UI | Mindestens Status-Filter als schnelle Toggle-Buttons | Lücke: **kein Status-Schnellfilter** im Formulare-Tab (z.B. „Nur Entwürfe", „Nur Live") |
| Pagination | Keine — alle Schemas auf einmal geladen | Pagination oder Virtual-Scroll bei >50 Einträgen | Kein Problem für Demo, wird aber bei echten Mandanten mit 50+ Formularen zur Performance-Last |

### 9c. Builder-Tiefe (vs. Markt)

| Feature | Ist | Tally/Typeform/JotForm | Lücke für DACH-KMU |
|---|---|---|---|
| Feldtypen | 10 Typen: text, textarea, email, select, radio, checkbox, date, number, file, consent | + rating/NPS, signature, phone, address, matrix/scale, image-choice, hidden field | Lücke: **Rating/NPS und phone** sind KMU-relevant (Kundenfeedback, Kontaktformulare). Signature wäre DACH-Arbeitsvertrag-relevant (aber komplex). |
| Conditional Logic | Vorhanden: 3 Operatoren (equals, not_equals, contains), 1 Quellfeld pro Zielfeld | Mehrfach-Bedingungen (AND/OR), Jump-to-Page-Logic | Lücke: **nur Single-Condition**, keine AND/OR-Kombinationen, kein Page-Jump |
| Validierung | Nur `required: true/false` | Mindest-/Höchstlänge, Regex-Pattern, min/max (number/date), custom error message | Lücke: **keine Feld-Validierungsregeln** außer Pflichtfeld. Für DACH-KMU relevant: PLZ-Format (5 Ziffern), Telefon-Pattern, IBAN-Plausibilität |
| Feldlabel-Formatting | Plaintext only | Markdown oder Rich-Text (fett, kursiv, Link) für Beschriftungen | Lücke: **keine Formatierung** in Feldbeschriftungen oder Formulartitel. Marktstandard bei langen Consent-Texten oder erklärenden Labels |
| Mehrseitigkeit | Vorhanden: Seitenumbrüche als Pseudo-Felder (`__page_break__`), Seitennavigation in Preview | Seitentitel, Fortschrittsbalken auf Ausfüll-Seite | Lücke: **kein Seitentitel pro Seite**, kein Fortschrittsbalken in der Vorschau |
| Vorlagen | 2 Vorlagen im Seed, kein Vorlagen-Browser | 20–50+ Vorlagen nach Kategorie | Lücke: **zu wenige Vorlagen**. DACH-relevante KMU-Vorlagen fehlen: Reklamation, Urlaubsantrag, Aufmaß, Schadensanzeige, Lieferantenerfassung |
| DSGVO | Consent-Feld vorhanden, Datenschutzerklärung-Link pro Feld konfigurierbar | Zweckbindung sichtbar, IP-Logging | Weitgehend vollständig. Lücke: **Datenverarbeitungszweck** ist nicht auf der Ausfüll-Seite sichtbar (erst wenn FD-4 kommt) |
| Danke-Seite / Nach-Submit | Keine — Preview hat disabled „Absenden"-Button | Konfigurierbare Danke-Seite mit Titel + Text + optionalem Redirect | Lücke: **keine Danke-Seite**-Konfiguration im Builder. Erst mit FD-4 sinnvoll |

### 9d. Submissions-Auswertung (vs. Markt)

| Feature | Ist | Markt (Typeform/JotForm) | Lücke |
|---|---|---|---|
| Stats-API | `GET /schemas/:id/stats` liefert: total, new, read, archived, thisWeek, thisMonth, averageCompletionRate (87% hardcoded) | Dasselbe + Conversion-Rate, Aufrufe | Weitgehend vorhanden. `averageCompletionRate` ist im MSW hardcoded (87%) — nicht aus echten Daten berechnet |
| Stats-Visualisierung | Nur drei Header-Kacheln (aktive Formulare, Eingänge/Woche, Completion-Rate) | Auswertungs-Dashboard pro Formular | Lücke: **keine Per-Formular-Auswertungsansicht** im UI — Stats-Endpoint wird gar nicht angezeigt |
| Feld-Analyse | Nicht vorhanden | Balkendiagramm pro Feld (select/radio), Freitext-Vorschau-Liste, Datumsverteilung | Lücke: **vollständig fehlend** (behandelt in FD-3 / FT-3) |
| Drop-off / Trichter | Nicht vorhanden | Seitenweise Abbruchrate bei mehrseitigen Formularen | Lücke: **vollständig fehlend** (Daten für Berechnung fehlen ebenfalls im Typ-Modell) |
| Submission-Filter | Status-Filter im MSW-Backend, aber **kein Filter-UI** | Status, Datum, Feld-Wert | Lücke: **kein Filter-UI** in der Submissions-Tabelle |
| Submission-Sortierung | MSW sortiert fix nach `submittedAt` absteigend, kein UI-Steuerelement | Feld + Richtung wählbar | Lücke: **kein Sortier-UI** |

### 9e. Moduleinstellungen

| Bereich | Ist | Soll | Lücke |
|---|---|---|---|
| `personal`-Sektion | Standard-Tab + Standard-Export-Format — beide wirken sich real aus | Vollständig | Vollständig |
| `tenant`-Sektion | Standard-Consent-Text, Datenschutz-URL, Benachrichtigung bei Eingang, Aufbewahrungsdauer | Vollständig | Weitgehend vollständig. `notifyOnSubmission` und `retentionDays` sind demo-stateful (kein Backend). Lücke: **kein Standard-Danke-Seiten-Text** (relevant wenn FD-0 Closed-Message einführt), kein Benachrichtigungs-E-Mail-Adresse-Feld |
| `ModuleSettingsShell` | Korrekt verwendet, `personal` + `tenant` Sektionen mit Icon | Korrekt | Vollständig |

### 9f. Edge-Cases und Robustheit

| Fall | Ist | Problem |
|---|---|---|
| Formular ohne Felder | Builder zeigt Leer-Zustand mit Hint. Preview zeigt Italic-Hinweis. Karte zeigt „0 Felder". | Kein Guard: Ein leeres Formular kann auf `active` gesetzt und „geteilt" werden (Dummy-Toast). Soll: „Veröffentlichen" nur wenn ≥1 Pflichtfeld oder DSGVO-Consent vorhanden. |
| Submission ohne `submittedBy` | Detail-Modal zeigt `--` korrekt. Tabelle zeigt `--`. | Korrekt |
| Sehr langer Formular-Titel | Karte truncated mit `truncate`. Modal-Titel truncated nicht (kein `truncate` auf `DialogTitle`). | `DetailModal` title ohne overflow-guard — langer Titel kann modal-header-Zeile brechen |
| Sehr viele Felder im Formular | Kein virtuelles Scrollen im Feld-Builder oder in der Detail-Feldliste. Bei 50+ Feldern würde die Seite scrollbar aber träge. | Kein unmittelbares Problem für Demo (max ~10 Felder in Seeds), aber für produktive Nutzung relevant |
| Submission mit fehlendem Feld-Mapping | `renderAnswer` fällt auf `String(value)` zurück wenn `field` nicht gefunden. | Korrekt — graceful degradation |
| i18n: ICU `{var}` statt `{{var}}` | Stichproben zeigen `{count}`, `{name}`, `{date}` korrekt in allen Formulare-Keys. | Kein akuter Fehler. Zu verifizieren via Screenshot-QA mit de/en/fr/it. |
| Status-Übergang ohne Guard | `active` Formular kann direkt auf `archived` gesetzt werden ohne Warnung (auch wenn noch offene Submissions). | Kein Guard vor dem Archivieren. Soll: Confirm-Dialog mit Hinweis auf offene Submissions. |
| Vorlagen-Karte: kein DetailModal | Vorlagen-Karten öffnen kein Detail-Modal — Klick auf die ganze Karte macht nichts, nur „Verwenden"-Button wirkt. | Projektweiter Standard (ganze Zeile / ganze Karte klickbar) verletzt. |

---

## 10. Modul-Tiefe-Phasen (FT-1 … FT-5)

> Diese Phasen bauen auf dem bestehenden Builder (F-1…F-5) und dem Distributions-Plan (FD-0…FD-5) auf. Sie adressieren die in Abschnitt 9 gefundenen Tiefe-Lücken. **Alle FT-Phasen sind rein FE-seitig (MSW) — kein Luke-Bedarf** außer dort explizit markiert.

### FT-1: Submissions-Vollständigkeit

**Was fehlt:** Kein Filter-UI, keine Sortierung, kein Guard vor Archivieren-ohne-Bestätigung, keine kontextgebundene Submissions-Ansicht am Formular-Detail-Modal.

**Akzeptanzkriterien:**

- Submissions-Tabelle (im Eingänge-Tab, pro Gruppe) hat einen Status-Schnellfilter: Toggle-Buttons „Alle / Neu / Gelesen / Archiviert" — direkt oberhalb der Tabelle, pro Gruppe separat. MSW-State bleibt unverändert, Filterung client-seitig.
- Submissions-Tabelle hat einen `SortMenu`-Eintrag: sortierbar nach „Eingegangen am" und „Absender" (auf- und absteigend). `shared/SortMenu` wiederverwenden.
- `ConfirmDialog` beim Archivieren eines Formulars, das noch `new`-Submissions enthält: „Dieses Formular hat {count} unbearbeitete Eingänge. Trotzdem archivieren?"
- Formular-Detail-Modal bekommt einen Tab „Eingänge": zeigt die Submissions dieses Formulars direkt (wiederverwendet `SubmissionsPanel`), ohne den Eingänge-Tab öffnen zu müssen. Zeilen-Klick öffnet wie gehabt das Submission-DetailModal.
- Vorlagen-Karten werden klickbar: Klick öffnet ein Detail-Modal mit Feldliste und „Verwenden"-Button im Footer (konsistent mit Formular-Detail-Modal).

**FE-mockbar:** ja — alles client-seitig oder aus bestehendem MSW-State.
**Luke-Bedarf:** nein.
**5-Phasen-Batch-Eignung:** hoch — in sich geschlossen, review-reif nach dieser Phase.

---

### FT-2: Builder-Tiefe

**Was fehlt:** Keine Feld-Validierungsregeln (nur `required`), keine Rating/NPS-Feldtypen, keine Seitentitel, kein DACH-KMU-Vorlagen-Pack, keine Danke-Seite-Konfiguration.

**Akzeptanzkriterien:**

- Feld-Konfig-Dialog (bestehend) bekommt optionale Validierungsregeln für `text`/`textarea`: min. / max. Länge (Zahl-Inputs). Für `number`: min. / max. Wert. Für `email`: bereits implizit durch `type="email"` — kein eigener Validator nötig. Für `text`: optionaler Pattern-Typ (Dropdown: „Frei", „PLZ (5 Ziffern)", „Telefon DACH", „IBAN"), der intern einen Regex-String speichert. Kein freies Regex-Feld (zu komplex für KMU-Nutzer).
- Neuer Feldtyp `rating`: Stern-Bewertung 1–5 (oder 1–10, konfigurierbar). Im Builder als Feldtyp wählbar, Preview zeigt Stern-Icons disabled. Im Submission-Detail wird Rating als Stern-Reihe gerendert.
- Seitentitel-Feld pro Seite: Beim Einfügen eines Seitenumbruchs erscheint direkt darunter ein editierbares Textfeld für den Seitentitel (Inline-Edit im Builder, gespeichert als `pageTitle`-Feld auf dem Seitenumbruch-Pseudo-Eintrag oder als eigenes Seiten-Metadaten-Array). Preview zeigt Seitentitel als `<h3>` über den Feldern.
- 5 neue DACH-KMU-Vorlagen im MSW-Seed: „Reklamationserfassung", „Urlaubsantrag (intern)", „Lieferantenbewertung", „Schadensanzeige", „Aufmaßerfassung Handwerk". Alle mit Consent-Feld.
- Danke-Seiten-Konfiguration am Schema: Neues optionales Feld `thankYouMessage` (Plaintext) + `redirectUrl` (URL, optional). Im Builder: eigener Abschnitt „Nach dem Absenden" unter den Feldern. Preview zeigt nach dem disabled „Absenden"-Button eine Vorschau der Danke-Nachricht.

**FE-mockbar:** ja — alle Änderungen am Builder und MSW-Seed.
**Luke-Bedarf:** `thankYouMessage` + `redirectUrl` müssen in das `FormSchema`-Backend-Modell + Migration (P2). Bis dahin nur im Frontend-Typ und MSW.
**5-Phasen-Batch-Eignung:** mittel — enthält mehr Substanz als FT-1, daher evtl. als eigener 5er-Batch.

---

### FT-3: Auswertungs-Dashboard

**Was fehlt:** Stats-Endpoint wird gar nicht visualisiert. Keine Feld-Analyse-Ansicht. Kein Drop-off/Trichter.

**Akzeptanzkriterien:**

- Formular-Detail-Modal bekommt einen dritten Tab „Auswertung" (neben „Details" und „Eingänge" aus FT-1).
- Auswertungs-Tab: Kennzahlen-Zeile oben (Gesamt-Submissions / Neue / Ø Bearbeitungszeit / Completion-Rate). Daten kommen aus `GET /schemas/:id/stats` (MSW-Endpoint bereits vorhanden).
- Pro Feld des Formulars wird eine Feld-Auswertungskarte gerendert:
  - `select` / `radio`: horizontales Balkendiagramm (count + Prozentsatz pro Option). Wiederverwendung: `ChartRenderer` aus berichte-Modul (E-1) falls dort bereits verfügbar, sonst simple CSS-Bars.
  - `checkbox`: Ja / Nein als zwei Balken.
  - `text` / `textarea`: Liste der 5 häufigsten nicht-leeren Antworten (Häufigkeit in Klammern). Bei mehr als 10 Antworten: „{count} Freitextantworten — in Eingänge-Tab ansehen".
  - `date`: Gruppiert nach Monat (Heatmap-Bar oder simple Monats-Leiste).
  - `number` / `rating`: Ø-Wert + Min/Max, einfache Verteilung.
  - `consent`: nur Prozentzahl Zustimmung (erwartet: 100%, Abweichungen sind DSGVO-relevant).
  - `file` / `email`: Anzahl ausgefüllter vs. leer gelassener Antworten.
- MSW: `GET /schemas/:id/stats` wird erweitert um ein Feld `fieldStats: Record<string, FieldStat>` (berechnet aus `SUBMISSIONS`-Array zur Laufzeit). `FieldStat` enthält: `total`, `empty`, `distribution` (Record<string, number> für Auswahl-Felder).
- Drop-off-Anzeige für mehrseitige Formulare: Trichterdiagramm (einfache CSS-Balken, abnehmend) wenn `pageCount > 1`. Daten sind im MSW simuliert (feste Prozentwerte pro Seite aus Seed), da echter Drop-off-Tracking Backend-Bedarf hätte.

**FE-mockbar:** ja — MSW kann Feld-Statistiken live aus SUBMISSIONS berechnen. Drop-off als Seed-Wert.
**Luke-Bedarf:** Für echten Drop-off-Tracking müsste Backend Seitenabbrüche erfassen (P3, post-Launch).
**5-Phasen-Batch-Eignung:** hoch — eigenständige UI-Schicht, review-reif als Ganzes.

---

### FT-4: Formulare-Tab Ordnung + Edge-Cases

**Was fehlt:** Kein View-Toggle (Grid/Liste), kein Status-Schnellfilter im Formulare-Tab, fehlende Guards (leeres Formular veröffentlichen, Archivieren mit Submissions), fehlender `closed`-Status (abhängig von Entscheidung #7).

**Akzeptanzkriterien:**

- Formulare-Tab Header: drei Status-Schnellfilter-Buttons „Alle / Entwürfe / Live / Archiviert" (Toggle, mutually exclusive außer „Alle"). Filterung client-seitig über `activeForms`-Array. Zähler auf Buttons.
- Grid/Liste-Toggle: Icon-Button Gruppe (2×2-Grid-Icon / Liste-Icon) analog zu anderen Modulen. Listen-Ansicht zeigt kompakte Tabelle (Titel, Status, Felder, Eingänge, Erstellt, Aktionen — `ItemActions`-Menü rechts). Grid bleibt Standard.
- Guard „Formular veröffentlichen": Wenn ein `draft`-Formular manuell auf `active` gesetzt wird (Archivieren-Toggle) und kein einziges Feld vorhanden ist: Warn-Toast „Dieses Formular hat keine Felder. Bitte erst Felder hinzufügen." und Status-Änderung blockiert.
- Guard „Archivieren mit offenen Submissions": `ConfirmDialog` mit Hinweis (aus FT-1, hier nochmals als Akzeptanzkriterium fixiert).
- DetailModal-Titel overflow: `truncate`-Klasse auf den Titel-Element des `DetailModal` (betrifft `shared/DetailModal` — Änderung global prüfen, ob andere Module davon profitieren).
- i18n-Screenshot-QA: alle vier Locales (de/en/fr/it) auf dem Formulare-Tab, Eingänge-Tab, Submission-Detail-Modal, Builder — Raw-Keys und doppelte `{var}` prüfen.

**FE-mockbar:** ja — vollständig.
**Luke-Bedarf:** nein.
**5-Phasen-Batch-Eignung:** sehr hoch — viele kleine in sich geschlossene Verbesserungen, ideal für Batch.

---

### FT-5: Settings-Vervollständigung + Vorlagen-Management

**Was fehlt:** Kein Standard-Danke-Seiten-Text in Settings (relevant nach FT-2), kein Benachrichtigungs-E-Mail-Adresse-Feld, kein Vorlagen-Management (eigene Vorlagen erstellen / pinnen).

**Akzeptanzkriterien:**

- Tenant-Settings bekommen zwei neue Felder: „Standard-Danke-Seite Nachricht" (Textarea, default leer) + „Benachrichtigungs-E-Mail" (E-Mail-Input, default leer — nur relevant wenn `notifyOnSubmission: true`). Beide demo-stateful im `formularePrefs`-Store.
- Personal-Settings bekommen ein neues Feld: „Standard-Formular-Ansicht" (Grid / Liste), damit der View-Toggle aus FT-4 einen persistenten Zustand hat.
- Vorlagen-Tab: „Als Vorlage speichern"-Aktion für bestehende Formulare (neuer Eintrag in `getFormActions`-Dropdown: „Als Vorlage speichern"). MSW setzt `isTemplate: true` via PATCH. Formular verschwindet aus Formulare-Tab, erscheint in Vorlagen-Tab.
- Vorlagen-Tab bekommt „Vorlage löschen"-Aktion (mit `ConfirmDialog`).
- Vorlagen-Detail-Modal (aus FT-1) bekommt im Footer zusätzlich „Bearbeiten"-Button (öffnet Editor) und „Löschen"-Button.

**FE-mockbar:** ja — vollständig.
**Luke-Bedarf:** nein für FE-Demo. Für Backend-Persistenz: `isTemplate`-Flag + `notifyEmail`-Tenant-Setting (P2, post-Launch).
**5-Phasen-Batch-Eignung:** mittel — setzt FT-2 (Danke-Seite) und FT-4 (View-Toggle) voraus.

---

## 11. Empfohlene 5-Phasen-Batches

### Empfehlung: Welche Phasen zuerst?

Die sinnvollste Reihenfolge für zwei aufeinanderfolgende 5-Phasen-Batches:

**Batch A (rein FE, sofort startbar, 5 Phasen):**

| Phase | Inhalt | Review-Reife |
|---|---|---|
| FD-0 | Lebenszyklus-Guards + Status-Modell | ja |
| FD-1 | Share-Dialog (echter Link + Clipboard) | ja |
| FT-1 | Submissions-Vollständigkeit (Filter, Sortierung, Guards, Vorlagen-Modal) | ja |
| FT-4 | Formulare-Tab Ordnung + Edge-Cases + i18n-QA | ja |
| FD-2 | Verteil-Übersicht + Kanal-Historie | ja |

**Rationale:** Alle fünf sind rein FE-seitig, kein Luke-Bedarf, jede Phase ist in sich review-reif. FD-0 muss vor FD-1 kommen (Guard: nur Live-Formulare teilbar). FT-1 gibt dem Modul die nötige Submissions-Tiefe. FT-4 bereinigt Edge-Cases und sorgt für saubere i18n-QA. FD-2 schließt die Verteilungs-Übersicht an den vollständigen Share-Dialog an.

**Batch B (FE-heavy, nach Batch A):**

| Phase | Inhalt | Review-Reife |
|---|---|---|
| FT-2 | Builder-Tiefe (Validierung, Rating, Seitentitel, Vorlagen, Danke-Seite) | ja |
| FT-3 | Auswertungs-Dashboard | ja |
| FD-3 | Antworten-Auswertung (Tab in FD-Struktur, überlappt mit FT-3) | merge mit FT-3 |
| FT-5 | Settings-Vervollständigung + Vorlagen-Management | ja |
| FD-4 | Öffentliche Ausfüll-Seite (Luke-Koordination, beginnt parallel) | Luke-Übergabe |

**Hinweis zu FD-3 / FT-3:** Diese beiden Phasen überlappen — beide zielen auf eine Feld-Auswertungs-Ansicht. Sie sollen **zusammengelegt** werden. Der Tab in Abschnitt 7 (FD-3) wird durch FT-3 ersetzt (konsolidierter Plan).

---

## 12. Backend-Bedarf für Luke (zusammengefasst, erweitert)

Alles in FD-0…FD-3 und **alle FT-Phasen** sind rein FE-seitig (MSW) und brauchen Luke **nicht**. FD-4 und FD-5 sind echter Backend-Bedarf. Zusätzlich gibt es zwei FT-Phasen mit optionalem spätem Backend-Bedarf.

| Aufgabe | Phase | Priorität |
|---|---|---|
| Tabelle `public_tokens` (`resource_type`, `resource_id`, `token`, `expires_at`, `password_hash`, `max_submissions`, `view_count`, `created_by`) | FD-4 | P0 |
| Öffentliche Gateway-Route-Gruppe `/public/` ohne `RequireAuthenticated` | FD-4 | P0 |
| `GET /public/forms/{token}` — Token validieren, Schema laden, HTML ausliefern | FD-4 | P0 |
| `POST /public/forms/{token}/submit` — Submission anlegen, keine JWT-Pflicht, Rate-Limit, Antwort-Limit prüfen | FD-4 | P0 |
| `GET /public/reports/{token}` — analog für berichte (gemeinsame Schicht) | FD-4/R-5 | P0 (gemeinsam planen) |
| `POST /api/v1/formulare/schemas/:id/share-links` — Token generieren, in `public_tokens` speichern | FD-4 | P1 |
| `GET /api/v1/formulare/schemas/:id/share-links` — Liste der Tokens pro Schema | FD-4 | P1 |
| `FormSchema`-Modell: `thankYouMessage`, `redirectUrl`, Validierungs-Regeln per Feld, `pageTitle` pro Seite | FT-2 (Backend-Ergänzung) | P2 |
| Seitenweise Drop-off-Tracking (neuer Submission-Event-Typ: Seite verlassen) | FT-3 (Backend-Ergänzung) | P3, post-Launch |
| Mailer: E-Mail-Versand mit personalisierten Link-URLs | FD-5 | P2 |
| QR-Code serverseitig (nur falls clientseitig abgelehnt) | FD-5 | P3 |
| Slug-System für eigene Link-Bezeichner | FD-5 | P3 |

**Wichtig:** Die `public_tokens`-Tabelle sollte **einmal gemeinsam** für formulare und berichte designed werden (nicht zweimal separat). Koordination Luke/Darien/Claude bevor FD-4 startet.

---

## 13. Verweise

- Bisheriger Formular-Builder (F-1…F-5): gemergt in `main`, `eb379a22`
- MSW-Handler: `desktop/src/renderer/src/mocks/handlers/formulare.ts`
- Typ-Modell: `desktop/src/renderer/src/api/formulare-types.ts`
- Backend-Routen (alle auth-gesichert): `backend/internal/gateway/route_formulare.go`
- Paralleles berichte-Vision-Dokument (Token-Schicht R-5): `.planning/berichte-report-authoring-VISION.md`
- Wiederverwenden: `components/shared/DetailModal`, `shared/SortMenu`, `ChartRenderer` (berichte E-1), `shared/RichTextEditor` (für Closed-Message-Feld)
- Modul-Tracker: `.planning/MASTER-TRACKER.md`
