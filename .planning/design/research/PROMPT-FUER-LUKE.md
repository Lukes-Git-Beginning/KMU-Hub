# Prompt fuer Luke — Update 2026-02-17 (Office-Strategie + Master-Plan)

Kopiere diesen Prompt und gib ihn Luke (oder seinem Claude/ChatGPT).

---

## DER PROMPT:

```
Hey Luke, wir haben heute zwei grosse Sachen fertig gemacht:

1. MASTER-PLAN.md — der komplette Frontend-Gesamtplan (174 Items, 14 Waves, ~55-62k LOC)
2. Office-Strategie — 3-Tier-Modell mit Collabora statt OnlyOffice

Bitte lies dir beides durch und gib Feedback.

## Was sich geaendert hat (gegenueber letztem Stand)

### AENDERUNG 1: Office-Strategie — 3 Tiers

Wir haben die Office-Integration komplett neu geplant:

| Tier | Preis | Lokales Office | Browser-Editor |
|------|-------|----------------|----------------|
| Starter (12 EUR) | Guenstig | Ja (immer) | Keiner |
| Business (19 EUR) | Mittel | Ja (immer) | TipTap (nur Texte) |
| Enterprise (25 EUR) | Premium | Ja (immer) | Collabora Online (Word+Excel+PPT) |

WICHTIG: "In Word oeffnen" ist in ALLEN Tiers dabei. Wer Office auf dem PC hat, kann immer damit arbeiten.

**Warum Collabora statt OnlyOffice:**
- OnlyOffice Community Edition ist AGPL — duerfen wir nicht kommerziell einbetten
- OnlyOffice Developer Edition: $1.950/Jahr, aber AGPL-Risiko bleibt bei Updates
- Collabora: MPL 2.0 (kommerziell sicher), weniger RAM (~1 GB vs ~2-4 GB), LibreOffice-basiert
- Collabora Business: ~1,82 EUR/User/Mo — wir verdienen trotzdem ~23 EUR/User bei Enterprise

**Was du dafuer bauen musst (Backend):**

1. **WebDAV-Server (alle Tiers):**
   - Damit Word/Excel direkt vom Server oeffnen/speichern koennen
   - Standard-Protokoll das Office nativ versteht
   - Kein Download/Upload noetig — Office arbeitet direkt auf dem Server
   - Versionierung: jede Speicherung = neue Version
   - Konflikterkennung bei gleichzeitiger Bearbeitung

2. **WOPI-Endpoints (Enterprise Tier, Collabora):**
   - 3 Endpoints: CheckFileInfo (GET), GetFile (GET), PutFile (POST)
   - Token-basierte Auth pro User+Datei
   - Collabora ruft diese Endpoints auf um Dateien zu lesen/schreiben
   - Collabora laeuft als Docker-Container (collabora/code, Port 9980)
   - Ressourcen: ~1 GB RAM + ~50 MB pro gleichzeitigem User

3. **Collabora Docker Setup:**
   - Docker-Container in docker-compose.yml einbinden
   - Reverse-Proxy konfigurieren (HTTPS, WebSocket)
   - Discovery-Endpoint fuer Frontend

**Lizenzen:**
- Collabora CODE (Development): Kostenlos, aber 10 Docs / 20 Connections — nur fuer Dev/Testing
- Collabora Business: ~1,82 EUR/User/Mo (bis 99 User)
- Collabora Enterprise: Individuell (Partner-Programm)
- ISV/Partner-Kontakt: collaboraonline.com/partner-programme/
- Self-Hosted-Kunden brauchen eigene Collabora-Lizenz

### AENDERUNG 2: MASTER-PLAN.md (Frontend-Gesamtplan)

Neues Dokument: `.planning/design/MASTER-PLAN.md` (~1800 Zeilen)

Fasst ALLES zusammen was im Frontend gebaut werden muss:
- 14 Waves, 174 Einzel-Items
- ~55.000-62.000 LOC geschaetzt
- ~75% Frontend-only, ~25% Backend-abhaengig
- Geschaetzte Timeline: 20-26 Wochen

**Deine Backend-Abhaengigkeiten nach Phase:**

| Deine Phase | Wave | Was du bauen musst |
|-------------|------|-------------------|
| Phase 8 (CRM) | Q2, 3 | Kontakt-API vereinheitlichen, Custom Fields, Firma als Entity, Deal-Pipeline |
| Phase 8 (Kommunikation) | 2 | Unified Inbox: Channel-Adapter, Message Queue, WebSocket |
| Phase 9 (Video+Wiki) | 2, 10, 11 | LiveKit Rooms/Tokens, Wiki-Versionierung |
| Phase 10 (E-Mail+Integration) | 2, 3, 5, Q4 | IMAP/SMTP, DATEV-Export, Bexio-Sync, PDF-Generierung |
| Phase 11 (Office+E-Signatur) | 11 | WebDAV-Server, WOPI-Endpoints, Collabora Docker, Skribble API |
| Phase 12+ (DSGVO+KI) | 13, Q5 | DSGVO-Datenabfrage/-loeschung, KI Embedding-Service |

### AENDERUNG 3: Deutschland-First

- EUR ist Standard-Waehrung (nicht mehr CHF)
- Deutsche MWSt-Saetze (19%/7%) als Default
- de-DE Locale als Default
- CH/AT kommen spaeter als Konfiguration dazu
- Frontend: formatCHF wird zu formatCurrency(amount, currency?)

### AENDERUNG 4: Theme-Cleanup (committed)

- Nature + Atelier Themes entfernt
- Jetzt 5 Themes: Cozy, Dreamy, Raumstation, Clean, Minimal
- desk-themes.ts und desk-asset-urls.ts bereinigt

## Was ich von dir brauche

1. **Lies MASTER-PLAN.md durch** — besonders die "[BACKEND-DEP]" markierten Items
2. **WebDAV-Server:** Hast du Erfahrung damit? Gibt es eine gute Go-Library?
3. **WOPI-Endpoints:** Sind das nur 3 REST-Endpoints — oder steckt mehr dahinter?
4. **Collabora Docker:** Hast du schon mit Collabora/LibreOffice Server gearbeitet?
5. **Sprint-Reihenfolge:** Was baust du als naechstes? Phase 8 (CRM) oder Phase 9 (Video)?
6. **Feedback:** Wo bist du anderer Meinung? Was fehlt? Was ist unrealistisch?

## Dateien

Alles committed auf `design/brainstorm`:
```bash
git fetch origin
git checkout design/brainstorm
```

Neue/geaenderte Dateien:
- `.planning/design/MASTER-PLAN.md` — DER Gesamtplan (174 Items, 14 Waves)
- `.planning/design/DESIGN-STATE.md` — Aktueller Status
- `.planning/design/ROADMAP.md` — Strategie-Roadmap (alle 7 Phasen DONE)
- `.planning/design/research/PROMPT-FUER-LUKE.md` — Dieser Prompt
- `docs/PRODUCT-STRATEGY.md` — Handoff-Dokument
- Alle Research-Dateien in `.planning/design/research/` (00-14)
```

---

## ANLEITUNG FUER DARIEN

1. Kopiere den Prompt oben (zwischen den ``` Markierungen)
2. Gib ihn Luke oder seinem Claude/ChatGPT
3. Luke kann den Branch pullen: `git fetch origin && git checkout design/brainstorm`
4. Alle Dateien sind committed und gepusht
