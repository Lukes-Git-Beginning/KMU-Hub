# Draft & Scheduled Deploy — Recherche-Report

> **Zweck:** Grundlage für das Draft/Deploy-Modell im Cosmi-Anpassungen-Editor.
> Darien-Entscheid (§0 EDITOR-VISION-BRIEFING.md): Speichern = Entwurf ODER geplantes Deploy an Tag X.
> Das ist Change-Management für Config: Draft → Review → geplantes Deploy → Live → Rollback.
>
> **Stand:** 2026-07-22. Quellen: Web-Recherche Session #25.

---

## 1 · ServiceNow Update Sets

**Quelle:** https://www.nowspectrum.com/blog/servicenow-update-sets-guide · https://sn-tricks.com/blog/servicenow-update-sets-complete-guide-migration-mastery/ · https://www.servicenow.com/community/itsm-forum/

### Mechanik

Update Sets sind **Container für Konfigurations-Änderungen**, die zwischen ServiceNow-Instanzen (Dev → Test → Prod) transportiert werden. Wichtig: sie erfassen Config-Records, **keine Daten**.

**Zustandsmodell (State Machine):**

```
[In Progress]  →  [Complete]  →  [Preview]  →  [Commit/Live]
    ↑                                ↓
 Änderungen                  Konflikt-Auflösung
 werden auto.                (Accept Remote / Accept Local / Merge)
 erfasst
```

- **In Progress:** Der aktive Update-Set. Alle Änderungen an der Instanz landen automatisch im aktiven Set (kein manuelles Auswählen).
- **Complete:** Developer markiert das Set als fertig — ab hier keine weiteren Änderungen.
- **Preview (Validation):** Auf der Ziel-Instanz: Preview prüft Konflikte, fehlende Dependencies, zirkuläre Referenzen. **Dieser Schritt ist Pflicht** — Überspringen ist die häufigste Ursache für Produktions-Incidents.
- **Commit:** Änderungen werden auf die Ziel-Instanz angewendet (live geschaltet).

**Export/Import:** Aus der Quell-Instanz als XML exportiert, in der Ziel-Instanz importiert. Kein direkter Push.

**Rollback:** Es gibt **keinen automatischen Rollback** nach Commit. Recovery = neues Update Set, das die Änderungen manuell rückgängig macht. Das macht Pre-Production-Testing zum kritischen Schritt.

### UX-Kern

- Klare Phasentrennung: Entwickler arbeitet in einer isolierten Instanz, nicht direkt in Prod.
- Preview-Screen zeigt Konflikte vor dem Commit — mit drei Auflösungsoptionen.
- Deployment-Flow: Export → Import → Preview → Commit (manuell, durch Person mit Zugriffsrechten auf die Prod-Instanz).
- **Keine Terminierung/Scheduling** nativ — wer wann committed, liegt bei der Person.

### Was übertragbar ist

- **Isolations-Prinzip:** Config-Änderungen in einer "Sandbox" sammeln, dann erst in die Prod-Schicht promoten. Direkt analog zur Cosmi-Draft-Overlay-Schicht.
- **Preview vor Commit:** Validierungsschritt, bevor die Änderung live geht. Für Cosmi: "Vorschau" der Änderungen im Editor-Kontext.
- **Konflikt-Auflösung:** Was passiert, wenn ein Label-Key bereits von vendor überschrieben ist? Cosmi braucht eine Antwort (in der Regel: tenant gewinnt, aber mit Provenance-Anzeige).
- **Was Cosmi anders machen muss:** Kein XML-Export. Kein manuelles Env-Hopping. Und — Kern-Unterschied — Cosmi braucht **echtes Scheduling** (Tag X wählen), das ServiceNow nicht hat.

---

## 2 · Salesforce Change Sets

**Quelle:** https://help.salesforce.com/s/articleView?id=platform.code_tools_changesets.htm · https://gearset.com/blog/how-to-deploy-changes-from-sandbox-to-production-in-salesforce/ · https://certifycrm.com/deploying-flows-to-production-with-change-sets/

### Mechanik

Change Sets sind Salesforce-native Container für Metadata (Felder, Flows, Apex, Layouts), die von einer Sandbox-Org zur Produktions-Org transportiert werden.

**Zustandsmodell:**

```
Sandbox-Org:
  [New/Draft]  →  [Upload]  →  →  →  →  →

Production-Org:
                              [Awaiting Deployment]  →  [Validating]  →  [Deployed]
```

- **Outbound Change Set (Sandbox):** Admin wählt Komponenten aus, klickt "Upload" → der Set wird zur Prod-Org übertragen.
- **Inbound Change Set (Production):** Set erscheint in der "Awaiting Deployment"-Queue. Erst jetzt kann jemand mit Prod-Zugriff deployen.
- **Validation:** Vor dem echten Deploy kann man "Validate" klicken — testet die Änderungen ohne sie zu committen (Apex-Tests laufen, Konflikte werden gemeldet).
- **Deploy:** Nach erfolgreicher Validation: "Deploy" committet alle Komponenten atomisch.

**Wichtige Einschränkungen (Salesforce-Community-Konsens):**
- Kein atomisches Rollback nach Deployment.
- Keine granulare Auswahl beim Deploy (alles oder nichts).
- Keine native Terminierung/Scheduling.
- Gearset (Third-Party) ergänzt: Dependency-Detection, Scheduling, Diff-Ansicht — was Salesforce nativ nicht hat.

### UX-Kern

- **Zwei-Org-Workflow:** Entwickler in Sandbox, Deploy durch Person mit Prod-Rechten. Klare Zugangstrennung.
- **Awaiting-Queue:** Der Prod-Admin sieht, was auf ihn wartet, und entscheidet, wann er deployed.
- **Validation-Step:** "Test without committing" — zeigt Fehler ohne Live-Wirkung. Entspricht dem Preview-Muster.
- **Keine Terminierung:** Salesforce empfiehlt Change Sets in Maintenance-Windows manuell zu deployen — das ist Workaround, kein Feature.

### Was übertragbar ist

- **Awaiting-Queue-Metapher:** Drafts, die deployed werden wollen, erscheinen in einer Queue. Für Cosmi: der Admin sieht im Anpassungen-Hub, welche Drafts "bereit zum Ausrollen" sind.
- **Validation-Step vor Live:** Cosmi-Pendant: Vorschau-Modus im Editor, bevor man "Übernehmen" klickt.
- **Zugangstrennung:** In Cosmi via RBAC — wer darf Drafts erstellen (alle mit `admin:customization:manage`), wer darf deployen (ggf. nur tenant:admin oder höher).
- **Was Cosmi anders macht:** Scheduling nativ eingebaut (nicht per Maintenance-Window-Workaround). Kein Org-Hopping. Und: Overlay-basiertes Rollback (Schlüssel entfernen = fällt auf daruntergehende Schicht zurück) statt kein Rollback.

---

## 3 · LaunchDarkly: Scheduled Rollouts + Approval Workflows

**Quelle:** https://launchdarkly.com/blog/launched-scheduling/ · https://launchdarkly.com/blog/launched-flag-approvals/ · https://launchdarkly.com/docs/home/releases/scheduled-changes · https://docs.launchdarkly.com/home/releases/workflows

### Mechanik

LaunchDarkly bietet drei aufeinander aufbauende Konzepte für kontrolliertes Deployment:

**A) Scheduled Flag Changes:**

```
[Flag-Änderung planen]  →  [Pending Changes Panel]  →  [Automatische Ausführung an Tag X]
```

- User plant eine Flag-Änderung für einen zukünftigen Zeitpunkt (Datum + Uhrzeit).
- Die geplante Änderung erscheint im **"Pending Changes"-Panel** (Sidebar der Flag-Targeting-Seite) — sichtbar für alle Teammitglieder.
- Zum Zeitpunkt X führt LaunchDarkly die Änderung automatisch aus.
- **Technisch:** "Semantic Patch"-Technologie — erfasst die Absicht der Änderung, nicht nur den Diff. Dadurch läuft die Änderung korrekt ab, auch wenn sich der Flag-Zustand zwischen Planung und Ausführung geändert hat.
- **Hinweis (2026):** Scheduled Changes sind in "maintenance mode" und werden zukünftig abgelöst durch dedizierte Progressive Rollout Features.

**B) Approval Workflows:**

```
[Änderung vorschlagen]  →  [Reviewer auswählen]  →  [Email-Benachrichtigung]
    ↓
[Review: Approve / Decline / Comment]  →  [Approved]  →  [Anwenden (Requester oder Reviewer)]
```

- Zustand: "Pending Review" → "Approved" → "Applied"
- **Requester kann eigene Requests nicht approven** (Vier-Augen-Prinzip).
- Approval und Application sind getrennt — man kann approven, und der Requester entscheidet erst später wann er anwendet.
- Enterprise: Approvals können **erzwungen** werden für bestimmte Environments (Production mandatory).

**C) Workflows (kombiniert):**

Progressive Rollout Workflow: 0% → 20% → 50% → 100% automatisch, jede Stufe terminiert.
Maintenance Window: Flag für definierten Zeitraum aktivieren, danach automatisch zurück.
Custom: beliebige Kombination aus Scheduling + Approvals.

### UX-Kern

- **Pending Changes Panel:** Immer sichtbar neben der Targeting-Ansicht — alle geplanten Änderungen auf einen Blick.
- **Approval-Request-Dialog:** Beschreibung + Reviewer-Auswahl in einem Schritt.
- **E-Mail-Benachrichtigung** an Reviewer und Requester bei jedem Statuswechsel.
- **Rollback:** Feature-Flag-Rollback = Toggle-Flip, wirkt in Millisekunden. Das ist LaunchDarkly's stärkstes Selling-Argument gegenüber Code-Deploys.

### Was übertragbar ist

- **Pending-Changes-Konzept:** Alle geplanten Änderungen eines Drafts auf einen Blick — auch Cosmi-Editor sollte eine Zusammenfassung "Was wird geändert" zeigen, bevor man deployt.
- **Scheduling-UX:** Datum + Uhrzeit wählen, Bestätigung, dann automatische Ausführung. Genau das braucht Cosmi für "terminiertes Deploy".
- **Semantic Patch / Intent-Capture:** Wichtige Einsicht: Cosmi-Drafts müssen **Absichten** speichern, nicht Snapshots. Wenn sich der vendor-Layer zwischen Draft-Erstellung und Deploy ändert, muss der Draft trotzdem sauber promoten.
- **Rollback = Overlay entfernen:** Analog zu LaunchDarkly — kein Code-Deploy, nur Daten-Operation. Augenblicklich wirksam.
- **Approval-Muster (später):** Für größere Tenants mit IT-Abteilung: Vier-Augen-Prinzip vor tenant-weitem Deploy. V1 nicht nötig, aber Architektur muss es erlauben.
- **Was Cosmi anders macht:** LaunchDarkly's Scheduling ist Feature-Flag-Granularität (eine Änderung pro Flag). Cosmi deployed einen ganzen Draft (Bundle von Änderungen an Feldern, Labels, Wertelisten) — näher am Release-Bundle-Konzept von Contentful Timeline.

---

## 4 · Contentful Timeline / Headless CMS Draft-Publish-Muster

**Quelle:** https://www.contentful.com/blog/introducing-timeline/ · https://www.sanity.io/glossary/drafts--publishing-workflow · https://github.com/payloadcms/payload/discussions/15125

### Mechanik

Headless CMS-Systeme (Contentful, Sanity, Payload CMS) sind der direkteste Analogon zu Cosmi's Config-Deployment, weil:
- Content = Daten (keine Code-Deploys)
- Draft → Publish ist Kernmuster
- Mehrsprachig (analog: mehrere Locales in Cosmi's i18n-Overrides)
- Tenant-Kontext (Content-Space = Tenant)

**Contentful Timeline — Zustandsmodell:**

```
[Ideation]  →  [Draft]  →  [In Review]  →  [Release]  →  [Published / Scheduled]
                                              ↑
                             (Bundle mehrerer Einträge = eine Welle)
```

- **Ideation:** Sicherer "try-out"-Raum, berührt nichts. Cosmi-Pendant: der Editor-Sandbox-Raum.
- **Draft:** Private Kopie im Bearbeitungs-Zustand.
- **Release:** Container, der mehrere Draft-Zustände bündelt — "eine Welle" (Campaign-Launch, Product-Update). Wichtig: entspricht dem Cosmi-Draft-Bundle (alle Felder + Labels + Wertelisten einer Überarbeitung zusammen).
- **Scheduled:** Release hat einen Deploy-Termin. Contentful führt automatisch zum Zeitpunkt X aus.
- **Published:** Live für alle.

**Sanity Draft/Publish:**
- Drafts sind private Kopien — nie sichtbar für End-User.
- Shareable Preview Links: Stakeholder können den Draft-Zustand sehen, ohne dass er live ist.
- Version Control: Rollback auf jeden früheren Stand.
- Scheduled Publishing: Pro Entry oder als Release-Bundle terminierbar.

**Payload CMS (technisch):**
- Diskussion über Vereinfachung des Draft/Publish-State-Machine: Zustände `draft | published | scheduled` — nicht mehr (kein `review`-Zustand in v4, der als separate Workflow-Step modelliert wird).
- Einfachste valide State Machine: **3 Zustände reichen für MVP.**

### UX-Kern

- **Preview vor Publish:** Shareable Preview Links — Stakeholder sehen den Zustand ohne Live-Schaltung.
- **Release-Bundle:** Nicht einzelne Änderungen schedulen, sondern ein kohärentes Paket. Stärker als "eine Änderung à la LaunchDarkly".
- **"Content exactly as it will appear at a specific point in time":** Vorschau-Link mit Zeitstempel — für Cosmi: Vorschau zeigt, wie das Modul nach dem Deploy aussieht.
- **Rollback:** Vorgänger-Version reaktivieren. In headless CMS typisch: jede Publish-Aktion erzeugt eine neue Version, Rollback = alte Version republizieren.

### Was übertragbar ist

- **3-Zustands-Modell (Draft/Scheduled/Live):** Minimal-State-Machine für MVP — genau richtig für Cosmi v1.
- **Release-Bundle-Konzept:** Ein Cosmi-Draft = Bundle aller Änderungen an einem Modul (Felder + Labels + Wertelisten zusammen). Nicht atomisch pro Schlüssel schedulen, sondern als Paket.
- **Preview-Link-Metapher:** Im Cosmi-Editor: "Vorschau öffnen" vor dem Deploy — zeigt das Modul mit den Draft-Overlays aktiv (genau das, was die Live-ICU-Mechanik + resolveConfig ermöglicht).
- **Scheduled Publish UX:** Datumsauswahl → Bestätigung → Countdown im Pending-Panel.

---

## 5 · Akamai Property Manager / AWS AppConfig — CDN/Infra-Config-Deployment

**Quelle:** https://techdocs.akamai.com/property-mgr/docs/how-activation-works · https://aws.amazon.com/blogs/mt/using-aws-appconfig-to-manage-multi-tenant-saas-configurations/ · https://aws.amazon.com/blogs/architecture/build-a-multi-tenant-configuration-system-with-tagged-storage-patterns/

### Mechanik (Akamai)

Akamai Property Manager ist das reifste Beispiel für versioniertes Config-Deployment mit Fast-Rollback:

**Zustandsmodell:**

```
[Version erstellen/editieren]  →  [Staging aktivieren]  →  [Production aktivieren]
                                         ↓ (Pending → Active)         ↓ (Pending Full Rollout → Active)
                                  2-3 Min / kein Traffic      3-15 Min / phased Rollout
```

- Jede Änderung erzeugt eine **neue Version** — die alte bleibt.
- Staging ≠ Production — explizite Aktivierung für jedes Netz.
- **Phased Rollout:** Auf Production wird in Phasen ausgerollt (erst die Traffic-Server, dann voller Rollout).
- **Fast Fallback** (innerhalb 60 Min): Ein-Klick auf die vorherige Version — 2-3 Minuten.
- **Manueller Rollback:** Alte Version reaktivieren (neue Aktivierung der alten Version-Nummer).

**AWS AppConfig:**
- Multi-Tenant: Distributed Model (pro Tenant-Account eigene Config-Ressourcen) oder Centralized Model (ein AppConfig-Set für alle Tenants, Feature-Flag-Ebene).
- **Deployment Strategies:** Linear oder Canary (schrittweise Rollout).
- **Automatic Rollback via CloudWatch:** Wenn Config-Änderung Fehlerrate steigen lässt → automatischer Rollback. Nicht relevant für Cosmi (kein Monitoring-Trigger benötigt).
- **Versionierung:** JSONB-Config mit Version-Feld + `isActive`-Boolean für Soft-Delete/Rollback.

### Was übertragbar ist

- **Versionierung als Fundament:** Jede Config-Änderung (jedes Deploy eines Drafts) erzeugt eine neue Version. Rollback = alte Version reaktivieren. Für Cosmi: `tenant_overlay`-Tabelle mit `version_id`, `deployed_at`, `rolled_back_at`.
- **Staged Rollout:** Nicht relevant für Cosmi v1 (kein Canary per User), aber die Architektur erlaubt es: Draft → Preview → Deploy an Tag X → optional Rollback.
- **Fast Fallback:** Cosmi-Rollback soll ebenfalls schnell sein — Overlay-Schlüssel entfernen oder vorherige Version reaktivieren, kein Code-Deploy.

---

## 6 · Staged/Scheduled Rollout — Allgemeine Best Practices

**Quelle:** https://xdi2.org/feature-flags-for-regulated-features-audit-approvals-and-rollback · https://docs.getunleash.io/guides/feature-flag-best-practices · https://learn.microsoft.com/en-us/azure/well-architected/operational-excellence/safe-deployments · https://brainhub.eu/library/deployment-strategies-explained

### Kernmuster aus der Industrie

**1. Draft/Preview/Live als Minimal-State-Machine:**
Drei Zustände reichen für das MVP. Mehr Zustände (In-Review, Approved, Staging, etc.) kommen on top wenn die Governance wächst — nicht von Anfang an.

**2. Audit Trail als Pflicht:**
Jede Config-Änderung loggt: wer, wann, was geändert, von welchem Zustand in welchen. Nicht optional — auch für Compliance (DSGVO-Tenant-Daten).
Minimales Audit-Event pro Deploy: `{ type, tenant_id, actor_id, version_from, version_to, scheduled_at, deployed_at, rolled_back_at }`.

**3. Rollback = Daten-Operation, kein Code-Deploy:**
Feature-Flag-Rollback in Millisekunden (LaunchDarkly). Overlay-Rollback in Sekunden (Schlüssel entfernen oder Version zurücksetzen). Nie einen Code-Deploy für Rollback benötigen.

**4. Nutzer-Ankündigung bei tenant-weiten Änderungen:**
Microsoft Business Central: Admin konfiguriert Notifications bei Tenant-Config-Änderungen.
Best Practice: Bei geplanten Änderungen optional eine In-App-Ankündigung für User anlegen können (z.B. "Am [Datum] werden Begriffe im Modul Kontakte angepasst").

**5. Approval Workflows für Enterprise:**
LaunchDarkly, SAP Cloud ALM, ServiceNow — alle reifen Produkte haben optionale oder erzwungene Approval-Schritte für Prod-Änderungen. In KMU-Kontext (Cosmi Zielgruppe): optional in v1, erzwingbar in v2.

**6. Inkrementelle Rollouts für Code-Deploys ≠ Config-Deploys:**
Canary/Blue-Green ist für Code. Config-Overlays sind per se instant und reversibel — der "Rollout" ist das kontrollierte Scheduling, nicht die schrittweise Ausdehnung auf User-Segmente.

---

## 7 · Empfohlenes Draft/Deploy-Modell für Cosmi

### 7.1 Zustandsmodell (State Machine)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    COSMI CUSTOMIZATION STATE MACHINE                 │
│                                                                      │
│  [keine Änderung]                                                    │
│       │                                                              │
│       ▼  Änderungen im Editor vornehmen                             │
│  [DRAFT]  ←──────────────── weiter bearbeiten ──────────────────┐  │
│       │                                                           │  │
│       ├─── "Jetzt übernehmen" ──────────────────────────────────►│  │
│       │         (sofortiger Deploy)                   [LIVE/ACTIVE] │
│       │                                                   │       │  │
│       └─── "Geplant für [Datum]" ─────► [SCHEDULED] ──►──┘       │  │
│                                              │                    │  │
│                                         Cron-Job                  │  │
│                                       an Tag X, 00:00             │  │
│                                                                   │  │
│  [LIVE/ACTIVE]  ──── "Rückgängig" ─────────────────────────────► │  │
│       │                 (Rollback auf                  [ROLLBACK/  │  │
│       │                  vorherige Version)            VORIGE VER.)│  │
│       │                                                            │  │
│       └─────────────────────────────────────────────────────────► ┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

**Zustände:**

| Zustand | Bedeutung | Sichtbarkeit |
|---|---|---|
| `draft` | Änderungen in Arbeit, noch nicht live | Nur im Editor sichtbar (Draft-Overlay aktiv im Editor-Kontext) |
| `scheduled` | Deploy-Termin gesetzt, wartet auf Ausführung | Admin sieht Countdown + Zusammenfassung |
| `live` | Overlay im tenant-Layer aktiv, alle User sehen die Änderungen | Alle User |
| `superseded` | Wurde durch eine neue Version ersetzt (vorheriger Live-Stand) | Archiv, für Rollback nutzbar |

**Übergänge:**

```
draft → live         : "Jetzt übernehmen" (sofort, kein Datum)
draft → scheduled    : "Geplant für [Datum]" (Cron-Job führt aus)
scheduled → live     : automatisch durch Cron-Job an Tag X
scheduled → draft    : Termin stornieren (zurück in Entwurf)
live → superseded    : sobald neue Version live geht
superseded → live    : Rollback (vorherige Version reaktivieren)
```

### 7.2 Technische Implementierung — Draft-Overlay-Schicht

**Erweiterte Overlay-Auflösung:**

```
effektiv (im Editor) = code_default ⊕ vendor ⊕ tenant_live ⊕ draft
effektiv (live für alle User) = code_default ⊕ vendor ⊕ tenant_live
                                   (= derzeit aktiver tenant-Layer)
```

Die `draft`-Schicht ist **nur im Editor-Kontext aktiv** — niemals für reguläre User.

**Datenmodell (MSW-first, später BE):**

```typescript
interface CustomizationDraft {
  id: string;                    // UUID
  tenant_id: string;
  module: string;                // z.B. "kontakte", "helpdesk"
  state: 'draft' | 'scheduled' | 'live' | 'superseded';
  changes: DraftChanges;         // sparse: nur was geändert wurde
  created_by: string;            // actor_id
  created_at: string;
  scheduled_at?: string;         // ISO 8601, nur wenn state === 'scheduled'
  deployed_at?: string;          // wann live gegangen
  rolled_back_at?: string;       // wann zurückgenommen
  superseded_by?: string;        // ID der Nachfolger-Version
  version: number;               // monoton steigend pro tenant/module
}

interface DraftChanges {
  fields?: FieldChange[];        // Custom Field Änderungen
  labels?: LabelChange[];        // Label-Override Änderungen
  value_sets?: ValueSetChange[]; // Value-Set Änderungen
}
```

**Warum sparse (nur Abweichungen):**
- Entspricht dem etablierten Overlay-Prinzip (§1 KONZEPT.md).
- Rollback = `superseded`-Version reaktivieren (ihr `changes`-Objekt neu applizieren) oder einfach die `live`-Version auf `superseded` setzen und die vorherige wieder auf `live`.
- Kein Full-Snapshot nötig — die Schichten darunter (vendor, code_default) bleiben unberührt.

### 7.3 Deploy-Mechanik: Sofort vs. geplant

**Sofort-Deploy ("Jetzt übernehmen"):**

```
Admin klickt "Übernehmen"
→ Validierung: Konflikte prüfen (z.B. Label-Key existiert in vendor?)
→ Draft-Changes werden in tenant_live-Overlay geschrieben
→ Draft-Record: state = 'live', deployed_at = now()
→ Alte live-Version: state = 'superseded'
→ Audit-Event: customization.deployed
→ (Optional) In-App-Banner für User: "Anpassungen wurden aktiviert"
```

**Geplantes Deploy ("Am [Datum] ausrollen"):**

```
Admin wählt Datum (DatePicker) + optional Uhrzeit (Default: 06:00 morgens)
→ Draft-Record: state = 'scheduled', scheduled_at = gewähltes Datum
→ Cron-Job läuft täglich (z.B. 06:00 UTC):
     SELECT * FROM customization_drafts
     WHERE state = 'scheduled' AND scheduled_at <= NOW()
     FOR UPDATE SKIP LOCKED;    -- concurrent-safe
   → Pro Treffer: identischer Flow wie Sofort-Deploy
→ Audit-Event: customization.scheduled_deployed
→ (Optional) User-Benachrichtigung wenn konfiguriert
```

**Cron-Job-Implementierung (Go-Backend):**
- Interval: täglich 06:00 UTC (außerhalb Kernarbeitszeit DACH)
- Idempotent: `FOR UPDATE SKIP LOCKED` verhindert Doppel-Ausführung
- Fehlerfall: Deploy schlägt fehl → Draft bleibt `scheduled`, Fehler ins Audit-Log, Admin bekommt Notification
- Entkoppelt: Cron-Job kennt nur `customization_drafts`-Tabelle, kein direkter Abhängigkeitsknoten

**Intent vs. Snapshot (LaunchDarkly-Einsicht für Cosmi):**
Der Draft speichert Absichten ("Label 'Kontakte' → 'Patienten'"), nicht den vollständigen tenant-Layer-Snapshot. Wenn zwischen Draft-Erstellung und Deployment der vendor-Layer eine Änderung erfährt, läuft der Draft trotzdem korrekt (er überschreibt nur seinen eigenen Schlüssel).

### 7.4 UX-Skizze — Deploy-Dialog

```
╔══════════════════════════════════════════════════════╗
║  Anpassungen übernehmen — Modul: Kontakte            ║
╠══════════════════════════════════════════════════════╣
║                                                      ║
║  Was wird geändert:                                  ║
║  ✓  3 Felder hinzugefügt (Versicherungsnummer, ...)  ║
║  ✓  1 Begriff geändert (Kontakte → Patienten)        ║
║  ✓  Deal-Phasen: 2 neue Stufen                       ║
║                                                      ║
║  Betrifft: alle 14 Nutzer deiner Organisation        ║
║                                                      ║
║  [○ Jetzt übernehmen]  [● Termin wählen]             ║
║                                                      ║
║  Termin: [  22. Juli 2026  ] um [  06:00  ]          ║
║          (in 3 Tagen, morgens vor Arbeitsbeginn)     ║
║                                                      ║
║  Nutzer-Ankündigung (optional):                      ║
║  [✓] In-App-Banner anzeigen, sobald Änderungen live  ║
║      gehen (Text: "Ihr Admin hat Cosmi angepasst")   ║
║                                                      ║
║         [Vorschau öffnen]   [Abbrechen]   [Bestätigen] ║
╚══════════════════════════════════════════════════════╝
```

**UX-Prinzipien:**
- **Was ändert sich:** Klare Zusammenfassung der Draft-Änderungen, keine technischen Keys.
- **Betroffene User:** Anzahl der Nutzer, die sofort betroffen sind (wichtig für KMU-Kontext: das ist nicht anonym, das sind 3–50 Menschen, die der Admin kennt).
- **Termin-Wahl:** DatePicker + optionale Uhrzeit. Default: morgens (06:00) am nächsten Werktag — analog zu Akamai's Rationale (außerhalb Kernarbeitszeit).
- **Vorschau:** Link/Button öffnet den Modul-Editor im Vorschau-Modus mit aktiven Draft-Overlays.
- **Ankündigung:** Optional, aber vorhanden — schlechtes Change-Management ist der häufigste Grund für User-Frustration bei tenant-weiten Änderungen (Microsoft Business Central-Erkenntnis).

### 7.5 Governance

**RBAC:**
- Bestehender Key: `admin:customization:manage` (§4 KONZEPT.md)
- Wer darf Drafts erstellen + bearbeiten: alle mit `admin:customization:manage`
- Wer darf deployen (Jetzt + geplant): dasselbe. In v1 keine Trennung (zu granular für KMU).
- Zukünftig (v2): eigener Key `admin:customization:deploy` — Entwickler/Konfiguratoren dürfen Drafts erstellen, nur IT-Admin darf deployen. Das Vier-Augen-Prinzip aus LaunchDarkly.

**Audit-Trail:**
Alle Deploy-Aktionen werden als Audit-Events gefeuert (Infra aus R-5):
```
customization.draft_created       { module, actor_id, timestamp }
customization.draft_updated       { module, actor_id, change_summary }
customization.deploy_scheduled    { module, actor_id, scheduled_at }
customization.schedule_cancelled  { module, actor_id }
customization.deployed            { module, actor_id, version, immediate: bool }
customization.rolled_back         { module, actor_id, version_from, version_to }
```

**Rollback:**
- UI: Im Anpassungen-Hub "Versionshistorie" pro Modul. Pro Version: Zeitstempel, Autor, Zusammenfassung. Rollback-Button reaktiviert die ausgewählte Version (setzt sie auf `live`, aktuelle auf `superseded`).
- Technisch: Overlay-Keys aus der alten Version neu applizieren ins tenant_live-Layer. Keine Datenverluste, kein Code-Deploy.
- Scope: Rollback ist modul-granular. Man kann Kontakte rollbacken, ohne helpdesk zu berühren.

**Nutzer-Ankündigung:**
- Option A (v1, simpel): In-App-Banner zum Deployment-Zeitpunkt. "Euer Admin hat Cosmi heute angepasst."
- Option B (v2): Vorankündigung bei geplanten Deploys. "Am [Datum] wird [Modul] angepasst. [Details]."
- Keine E-Mail-Notification in v1 (zu viel Infra). Nur In-App-Banner.

---

## 8 · MVP vs. Später

### V1 — Was ins MVP gehört

**Ziel:** Kein Over-Engineering. Das MVP löst das reale Problem: ein Admin macht Änderungen und will sie kontrolliert ausrollen, nicht versehentlich live schalten.

| Feature | Begründung |
|---|---|
| **Draft-Zustand** | Kernbedürfnis: "als Entwurf speichern". Ohne Draft kein kontrollierter Rollout. |
| **"Jetzt übernehmen" (sofort-Deploy)** | Einfachster Rollout-Pfad. Reicht für 90% der Fälle in v1. |
| **"Geplant für [Datum]" (Scheduled Deploy)** | Darien-Entscheid (§0 EDITOR-VISION-BRIEFING). Unterscheidungsmerkmal vs. allen Wettbewerbern. Technisch einfach (Cron-Job). Direkt in v1. |
| **Vorschau im Editor** | Vor dem Deploy sehen, was live geht. Bereits durch ICU-Overlay-Mechanik möglich. |
| **Änderungs-Zusammenfassung im Deploy-Dialog** | "Was ändert sich" muss sichtbar sein — ohne das ist das Scheduling-Feature blind. |
| **Rollback auf vorherige Version** | Minimal-Safety-Net. Ohne Rollback ist ein Admin zu ängstlich, Änderungen zu planen. Technisch: Overlay-Version reaktivieren. |
| **Audit-Trail (deploy-Events)** | Pflicht. Wer hat wann was deployed? Für Support, Compliance, Debugging. |
| **In-App-Banner bei Deployment** | Minimale Nutzer-Kommunikation — ein Banner reicht in v1. |

**Was NICHT in v1:**
- Approval Workflow / Vier-Augen-Prinzip (zu komplex für KMU-v1, wo Admin = IT-Abteilung)
- Nutzer-Vorankündigung per E-Mail
- Modul-übergreifende Multi-Modul-Drafts (z.B. "Kontakte + Helpdesk zusammen deployen")
- Versionierungs-Diff-Ansicht (welche Felder genau geändert von wo nach wo — kommt in v2)
- Canary/Staged Rollout auf User-Segmente (für SaaS-Config nicht relevant, alle User bekommen dieselbe Config)

### V2 — Nächste Stufe

| Feature | Wann sinnvoll |
|---|---|
| Approval Workflow (Vier-Augen) | Wenn erste Enterprise-Kunden kommen, die IT-Admin ≠ Konfigurator haben |
| Vorankündigung geplanter Änderungen | Nach v1-Feedback: "ich hätte das gerne früher gesehen" |
| Multi-Modul-Draft-Bundle | Wenn Admin sagt "ich möchte Kontakte + Helpdesk zusammen umbenennen" |
| Detaillierte Versions-Diff-Ansicht | Wenn Audit-Trail-Daten wachsen und Admin historisch vergleichen will |
| Scheduled Deploy mit Uhrzeit (nicht nur Datum) | Uhrzeit-Genauigkeit für internationale Teams mit Service-Windows |
| E-Mail-Notification | Wenn Tenant-User-Base wächst und In-App-Banner nicht mehr reicht |

---

## 9 · Zusammenfassung — Entscheidungs-Tabelle

| Dimension | Cosmi-Entscheid (empfohlen) |
|---|---|
| **Minimale Zustände** | `draft` → `scheduled` → `live` → `superseded` |
| **Sofort-Deploy** | Ja, in v1 |
| **Scheduled Deploy** | Ja, in v1, via Cron-Job (täglich 06:00 UTC) |
| **Cron-Mechanik** | `FOR UPDATE SKIP LOCKED` + idempotent + Fehler→Audit |
| **Draft-Granularität** | Pro Modul (nicht pro Feld/Schlüssel) |
| **Speicher-Modell** | Sparse/Intent (nur Abweichungen), nicht Full-Snapshot |
| **Rollback** | Overlay-Operation, kein Code-Deploy, modul-granular |
| **Approval v1** | Kein Approval-Workflow (zu granular für KMU) |
| **Audit** | Pflicht, alle Deploy-Events, via R-5-Infra |
| **Nutzer-Ankündigung v1** | In-App-Banner zum Deploy-Zeitpunkt |
| **Rollback-UX** | Versionshistorie im Hub, ein Klick reaktiviert |

---

**Quellen:**
- ServiceNow Update Sets Guide: https://www.nowspectrum.com/blog/servicenow-update-sets-guide
- ServiceNow SN-Tricks Guide: https://sn-tricks.com/blog/servicenow-update-sets-complete-guide-migration-mastery/
- Salesforce Change Sets Help: https://help.salesforce.com/s/articleView?id=platform.code_tools_changesets.htm
- Gearset Sandbox-to-Production: https://gearset.com/blog/how-to-deploy-changes-from-sandbox-to-production-in-salesforce/
- LaunchDarkly Scheduling Blog: https://launchdarkly.com/blog/launched-scheduling/
- LaunchDarkly Flag Approvals Blog: https://launchdarkly.com/blog/launched-flag-approvals/
- LaunchDarkly Scheduled Changes Docs: https://launchdarkly.com/docs/home/releases/scheduled-changes
- LaunchDarkly Workflows Docs: https://docs.launchdarkly.com/home/releases/workflows
- Contentful Timeline: https://www.contentful.com/blog/introducing-timeline/
- Sanity Draft/Publish Glossary: https://www.sanity.io/glossary/drafts--publishing-workflow
- Payload CMS Draft/Publish Discussion: https://github.com/payloadcms/payload/discussions/15125
- Akamai Property Manager Activation: https://techdocs.akamai.com/property-mgr/docs/how-activation-works
- AWS AppConfig Multi-Tenant: https://aws.amazon.com/blogs/mt/using-aws-appconfig-to-manage-multi-tenant-saas-configurations/
- AWS Multi-Tenant Config Architecture: https://aws.amazon.com/blogs/architecture/build-a-multi-tenant-configuration-system-with-tagged-storage-patterns/
- Feature Flags für Regulated Features: https://xdi2.org/feature-flags-for-regulated-features-audit-approvals-and-rollback
- Unleash Feature Flag Best Practices: https://docs.getunleash.io/guides/feature-flag-best-practices
- Microsoft Business Central Tenant Notifications: https://learn.microsoft.com/en-us/dynamics365/business-central/dev-itpro/administration/tenant-admin-center-notifications
- Azure Well-Architected Safe Deployments: https://learn.microsoft.com/en-us/azure/well-architected/operational-excellence/safe-deployments
