# Auslieferungsmodell — Web, Kundenserver, Modul-Zuschnitt

> Stand 2026-08-12. Grundlage: Lukes `launch-lagebild-2026-08-12.md` plus eine Arbeitsrunde
> Darien ↔ Claude, in der die Vorschläge gegen den Code geprüft wurden. Alles mit Pfad:Zeile ist
> nachgeschaut, nicht geschätzt. Kein Produktionscode angefasst.
>
> **Zweck: Vorschlag zur Diskussion — nichts davon ist entschieden.** Das Dokument trennt zwei
> Sorten von Aussagen, und die Unterscheidung ist wichtig:
>
> - **Befunde** (Abschnitt 2, 3, 7) sind am Code gemessen. Sie gelten unabhängig davon, wofür
>   sich die Runde entscheidet.
> - **Vorschläge** (Abschnitt 1) sind Dariens Vorschläge. Abschnitt 4 bis 6 rechnen durch, was
>   folgen *würde*, wenn die Runde ihnen zustimmt.

---

## 1 · Zur Diskussion gestellt — Dariens Vorschläge

**Nichts hiervon ist beschlossen.** Es sind Vorschläge, die in der Runde angenommen, verworfen
oder geändert werden können. Alles Folgende rechnet lediglich durch, was aus ihnen folgen würde.

1. **Kein Multi-Tenant-SaaS für Piloten — Vorschlag:** jeder Kunde bekommt einen **eigenen
   Hetzner-Server**. Begründung: geschlossenes System beim Kunden, Zentria greift nur mit
   erteilter Berechtigung zu.
2. **Erster Pilot unentgeltlich**, Zugang für Zentria vorab abgesprochen.
3. **ORBIT vorerst zurückstellen** — Hardware, kommt später.
4. **Umfangreiche, unfertige Module aus dem 1.0 nehmen** und nach dem Piloten fertig bauen.
   Welche genau: offen.
5. **Editor-Rollout nicht mit `finanzen` starten**, sondern mit Modulen, die sicher im 1.0
   bleiben — Kalender, Dokumente, evtl. Schichtplanung. Deckt sich mit Lukes Empfehlung.
6. **Website und Preisgestaltung** in den nächsten 1–2 Tagen separat überarbeiten.
7. **Nur gebuchte Module auf den Kundenserver** — nicht ausliefern und abschalten. Wörtlich:
   die BMW-Sitzheizung ist kein Vorbild.

> Vorschlag 1 ist der folgenreichste: An ihm hängen Abschnitt 3 bis 5 vollständig. Fällt er,
> fällt der größte Teil des neuen Aufwands mit ihm weg — dann gelten Lukes ursprüngliche Zahlen.

---

## 2 · Die Kernfrage: Web oder Desktop

Lukes Entscheidung 2 lautete „Web zuerst, Desktop später". In der Runde kam die Rückfrage, was das
überhaupt heißt — Cosmi ist schließlich ein Programm. Der Reihe nach:

### Was heute schon gilt

**Die `.exe` ist kein lokales Programm.** Sie ist eine Hülle um dieselbe React-App, die per HTTP
mit dem Backend spricht (`desktop/src/renderer/src/lib/constants.ts:4`). Alle Daten liegen auf dem
Server. Ohne Serververbindung tut der Client nichts.

Daraus folgt für die geäußerten Sorgen:

| Sorge | Befund |
|---|---|
| Latenz wird schlechter | **Nein.** Jede Liste ist heute schon ein Netzwerkaufruf. Unterschied ist allein das erstmalige Laden der App |
| Einstellungen/Auswahl/Anpassen fühlt sich träger an | **Nein.** Reines React im Client, identischer Code |
| Rollen und Zugriffe müssen genauso funktionieren | **Tun sie.** `usePermissionsStore.initFromServer()` — die UI fragt den Server, was der Benutzer darf. Serverseitig, unverändert |
| Mandantenzuordnung | **Unverändert.** JWT mit `tid`-Claim, ausgelesen in `backend/internal/middleware/auth.go:59`, durchgesetzt per RLS |

### Was im Web tatsächlich anders wäre

| Was | Konsequenz |
|---|---|
| Eigene Fensterleiste (`hooks/useElectronIPC.ts`) | fällt weg, Browser-Chrome stattdessen |
| Token-Speicherung über Electrons verschlüsselten Speicher (`stores/auth.ts`) | muss anders gelöst werden — der einzige echte Brocken |
| Editor als eigenes OS-Fenster (`src/main/ipc/editor-window.ts`) | wäre ein Tab. Dariens Vorschlag: auf zweitem Bildschirm ausreichend — zu bestätigen |
| Screen-Share-Quellenauswahl (`features/video/ScreenSourcePicker.tsx`) | entfällt, **wenn** das Meeting-Modul nicht im 1.0 ist (Vorschlag 4) |
| Desktop-Benachrichtigungen | gehen im Browser, mit Erlaubnisdialog |

Werden die beiden Vorschläge angenommen, schrumpft der Umbau von 15 betroffenen Renderer-Dateien
auf im Wesentlichen **eine** (Token-Speicherung) plus die Fensterleiste. Ohne sie bleiben Editor-
Fenster und Screen-Share als echte Posten stehen.

### Zum Gegenargument aus dem Lagebild

Das Lagebild warnt, die Sandbox-Mechanik des Editors hänge an Electron. **Das ist zu pessimistisch:**
`modules/admin/anpassungen/editor/customization-sync.ts` nutzt **`BroadcastChannel`** (Vorschau ↔
Hauptfenster) und **`localStorage`** (Entwurfsübergabe an ein startendes Fenster) — beides
Web-Standards, im Browser identisch. Electron-spezifisch ist nur das Öffnen und Schließen des
Fensters. Der Editor ist kein Blocker für den Web-Weg.

### „Fühlt sich weniger fertig an" — die Antwort heißt PWA

Berechtigte Sorge, aber sie zielt auf Browser-Chrome, nicht auf Leistung. Eine installierbare PWA
läuft in einem eigenen Fenster ohne Adressleiste, mit eigenem Icon in der Taskleiste, per
Doppelklick startbar. PWA ist für Mobile ohnehin der geplante Weg (Phase E).

---

## 3 · Was „eigener Server pro Kunde" am Code ändern würde

> Dieser Abschnitt rechnet **Vorschlag 1** durch. Die Codebefunde selbst gelten unabhängig davon;
> was daraus folgt, hängt daran, ob die Runde dem Vorschlag zustimmt.

### 3.1 Die Serveradresse ist heute fest einkompiliert

```
constants.ts:4 →  API_BASE_URL = import.meta.env.RENDERER_VITE_API_URL || 'http://localhost:8080'
```

`import.meta.env` wird **beim Bauen** eingesetzt, nicht beim Start gelesen. Bei einem Server pro
Kunde bedeutet das: **pro Kunde ein eigener Installer-Build**, bei jedem Update neu bauen, neu
signieren, neu verteilen — mal Anzahl Kunden.

Zwei Auswege: Serveradresse zur Laufzeit konfigurierbar machen (ein Build für alle), oder Web
(die Oberfläche kommt vom Server des Kunden und kennt ihre Adresse automatisch).

**Das wäre unabhängig von Entscheidung 2 nötig** — Desktop wie Web, ein Kunde oder zehn. Steht in
keiner bisherigen Planung.

### 3.2 Es gibt keine Registry — der Server baut aus dem privaten Repo

`cd.yml` läuft auf einem self-hosted Runner **auf der Produktionsmaschine**; `deploy.sh` macht dort
`git pull` und baut die Container lokal. In `docker-compose.yml` kommen nur Fremd-Images fertig
(Redis, MinIO, LiveKit, OnlyOffice) — die 24 eigenen Dienste entstehen auf der Maschine.

Für einen Server ist das elegant. Für Kundenserver bricht es zweimal:

1. **Jede Kundenmaschine bräuchte Zugriff auf euren Quellcode** (Deploy-Key aufs private Repo).
2. **Jede baut alles selbst** — Build-Fehler passieren beim Kunden statt bei euch.

**Vorschlag:** einmal zentral bauen, Images in eine Registry (ghcr.io), Kundenserver ziehen nur
fertige Artefakte. Update = `docker compose pull && up -d`. Nebeneffekt: Versionen pro Kunde
festhaltbar — Pilot einfrieren, während weiterentwickelt wird.

### 3.3 Modul-Schaltung: was existiert, was nicht

Mehr ist gebaut als erwartet:

| Baustein | Wo | Stand |
|---|---|---|
| Modul-Aktivierung pro Mandant | `tenant_module_activations` (Migration 000250) | da, unter RLS |
| Vertragsdaten | `tenants.plan_type` / `support_tier` / `subscription_status` / `seat_limit` | da |
| Verwaltungsoberfläche | `modules/admin/license/LicenseAdminHubTab.tsx` | da |
| Branchenprofile | `config/business-profiles.ts` — 10 Branchen mit `defaultModules` / `optionalModules` | da, ausdrücklich für Onsite-Konfiguration gebaut |
| Modulzuweisung pro Mitarbeiter | `user_module_grants` (Migration 000220) | da |
| Echtes Abschalten | Env-Var `COSMI_MODULE_<NAME>_ENABLED` pro Server | da |

Die Einschränkung steht in der Migration selbst:

> „Activation is bookkeeping, not enforcement: nothing in this migration blocks a request to a
> deactivated module. The deployment-wide `modules.*` feature flags keep doing that job."
> — `000250_tenant_license_and_subscription.up.sql:10`

Die Tabelle sagt also nur, was gebucht ist; abgeschaltet wird über Env-Vars, und die gelten **pro
Server**. Für klassisches SaaS wäre das zu grob — **bei einem Server pro Kunde wäre es genau die
richtige Ebene.** Was für SaaS noch zu bauen wäre, existierte für dieses Modell bereits.

### 3.4 Modul weglassen statt abschalten — ginge, und zwar richtig

Der Einwand hinter Vorschlag 7 ist technisch berechtigt: Auf einer Maschine, auf der der Kunde Root
hat, ist ein Flag keine Zugangskontrolle, sondern eine Bitte. **Die Architektur trüge echtes
Weglassen bereits:**

- **Backend:** In `docker-compose.yml` ist **jedes Modul ein eigener Dienst** — `schichten`,
  `fuhrpark`, `vermietung`, `helpdesk`, `wiki`, `berichte`, `formulare`, `inventar`, `einkauf`,
  `produktion`, `vertraege`, `rapporte`, `dialer`. Ein Compose-File ohne den Dienst heißt: das
  Image kommt nie auf die Maschine. Der Code ist dort **physisch nicht vorhanden**.
- **Frontend:** **82 `lazy()`-Importe** im Renderer — jedes Modul landet beim Bauen in einer eigenen
  Datei. Beim Ausliefern lässt man die Dateien ungebuchter Module weg.

**Und hier dreht sich das Desktop-Argument um:** Eine `.exe` bringt immer alle Module mit, egal was
gebucht ist — das ist die Sitzheizung, buchstäblich. Im Web liefert der Server nur aus, was
existiert. Der Anti-BMW-Anspruch ist im Web sauberer umsetzbar als im Desktop.

Was nicht verschwindet, ehrlich: Das Gateway kennt alle Routen (trifft aber ins Leere, kein Dienst
dahinter), und die Migrationen legen alle Tabellen an (leer). Beides Hüllen ohne Inhalt.
**Migrationen sollten pro Kunde nicht beschnitten werden** — sie bauen aufeinander auf, und ein
Kunde mit Schema-Lücken wird beim ersten Nachbuchen zum Sonderfall.

**Formulierbare Zusage, falls so entschieden:** *„Module, die Sie nicht gebucht haben, sind auf
Ihrem Server nicht installiert."* Wörtlich wahr und prüfbar — passt zur
Datensouveränitäts-Erzählung.

### 3.5 Nachbuchen später

Billig: Env-Var setzen, Dienst ergänzen, Gateway neu starten, Haken in der Aktivierungstabelle.
Kein neuer Build, kein Update beim Kunden. Der Gateway prüft ohnehin beides — das Flag ist die
Obergrenze, die Tabelle das Feintuning darunter (`gateway/route_settings.go:885`).

---

## 4 · Was sich an Lukes Etappenplan verschieben würde

> Alles in diesem Abschnitt gilt **nur, wenn die Vorschläge angenommen werden.** Bleibt es beim
> Installer und beim gemeinsamen Server, gelten Lukes Zahlen unverändert.

### Fiele weg

| Posten | Grund | Bedingung |
|---|---|---|
| G0-11 Installer nie gebaut | entfällt bei Web | Web statt Desktop |
| G0-12 Code-Signatur | entfällt bei Web (spart 150–750 €/Jahr) | Web statt Desktop |
| Kein Update-Kanal (G2, 1,5 PT) | Server aktualisieren reicht | Web statt Desktop |
| Screen-Share-Umbau | Meeting-Modul nicht im 1.0 | Vorschlag 4 |
| Editor-Fenster-Problem | Tab auf zweitem Bildschirm | Vorschlag zu bestätigen |

### Käme neu dazu — steht in keiner Analyse

Lukes Streichliste sagt: *„ORBIT/Self-Hosted nicht kundenfähig machen — 4 PT, verkauft sich erst,
wenn jemand danach fragt."* **Bei „eigener Server pro Kunde" wäre dieser Posten kein
Streichkandidat, sondern Voraussetzung.** Technisch ist es dasselbe Problem wie ORBIT — mehrere
Deployments, die nicht der Hauptserver sind. Der Unterschied ist nur, wem die Maschine gehört.

Das ist das stärkste Gegenargument zu Vorschlag 1 und gehört offen auf den Tisch: Ein Server pro
Kunde kauft Abgrenzung und zahlt sie mit Betriebsaufwand.

| Neu | Aufwand |
|---|---|
| Container-Registry | Teil der 3–5 PT |
| Serveradresse zur Laufzeit | dito |
| Einrichtungs-Erzeuger (Compose + Dateiliste aus der Modul-Liste) | dito |
| Auslieferung ohne ungebuchte Module | dito |
| **Summe** | **~3–5 PT** |

Nicht die vollen 4 PT aus dem Lagebild — Ansible und Compose existieren. Aber auch nicht null.

### Würde teurer als gedacht

Diese G1-Posten skalierten dann mit der Kundenzahl:

- **`restore.sh` ist strukturell kaputt und nie ausgeführt** — dann pro Kundenmaschine relevant
- **Backups nur lokal, unverschlüsselt, auf derselben Platte** — mal Anzahl Kunden
- **Kein Alarm bei stillem Backup-Fehlschlag** — genau die Lücke, durch die der MinIO-Ausfall zwei
  Monate unbemerkt blieb
- **`RUNBOOK.md` ist ein Skelett**, SSH-Key und Vault-Passwort existieren nur bei Luke

Lukes „blinder Fleck" — was passiert, wenn der erste Kunde anruft und Luke im Hauptjob sitzt —
würde durch mehrere Maschinen schärfer. Bei einem unentgeltlichen Piloten tragbar; ab dem zweiten
Kunden nicht mehr.

---

## 5 · Blocker-Stand, wenn die Vorschläge angenommen werden

### Unverändert offen (G0)

| Blocker | PT | Hängt an |
|---|---|---|
| **G0-6 Kontakte verwirft 9 Formularfelder beim Speichern** | 2,5 | nichts — Kontakte ist immer dabei |
| G0-1…G0-5 Website, Impressum, /ki, AVV, Beta-Zahlen | 3,25 | Website-Runde (1–2 Tage) |
| G0-7 Helpdesk-Zuweisung scheitert am echten Backend | 0,5 | ist Helpdesk im 1.0? |
| G0-8 E-Rechnungs-Knopf meldet Erfolg ohne Wirkung | 1 | ist Finanzen im 1.0? |
| G0-9 Fuhrpark zeigt rohe Übersetzungsschlüssel | 0,25 | ist Fuhrpark im 1.0? |
| G0-10 `booking.zentria.tech` existiert nicht | 0,15 | **Kalender — Editor-Startmodul** |
| G0-13 Pilot-0-Doku beschreibt den ZFA-Piloten | 0,25 | trivial |

**G0-6 ist der härteste Posten**, weil er von keiner Modul-Entscheidung berührt wird: Kontakt
anlegen, Adresse und Mobilnummer ausfüllen, speichern, neu laden — neun Felder weg, ohne
Fehlermeldung. Das ist der Moment, der in einer Live-Demo passiert.

**Drei Blocker lösten sich von selbst auf** (G0-7, G0-8, G0-9), wenn Helpdesk, Finanzen oder
Fuhrpark aus dem 1.0 fielen. Deshalb der Vorschlag, die Modul-Streichliste vor allem anderen zu
entscheiden — sie bestimmt, welche Blocker überhaupt noch welche sind.

### Grobe Rechnung bis „zeigbar" — unter den Vorschlägen dieses Dokuments

| | PT |
|---|---|
| G0 unverändert | ~7,9 |
| minus wegfallende Modul-Blocker (je nach Streichliste) | −0,25 bis −1,75 |
| plus Auslieferungsweg (nur bei Server pro Kunde) | +3 bis 5 |
| **Summe** | **~10–13** |

Lukes Etappen 0–2 lagen bei 8–12 PT gegenüber 12–16 verfügbaren Tagen. **Wäre weiter machbar, aber
ohne Puffer** — der Auslieferungsweg kostete ungefähr das, was Installer und Signatur einsparen.

Nicht enthalten: G1 (~10 PT, vor dem ersten echten personenbezogenen Datensatz). Bei einem Piloten
mit echten Kundendaten würde davon mindestens Backup/Restore scharf — unabhängig davon, ob er
unentgeltlich ist.

---

## 6 · Was die Runde entscheiden muss

Die Vorschläge aus Abschnitt 1 stehen hier als Fragen. Keine davon ist vorentschieden.

1. **Modul-Streichliste für 1.0.** Was bleibt, was kommt nach dem Piloten. Bestimmt drei G0-Blocker
   und den Zuschnitt jeder Kundeninstallation. Vorschlag: **zuerst**, weil vieles daran hängt.
2. **Ein Server pro Kunde oder gemeinsames SaaS?** Der folgenreichste Punkt. Dafür: geschlossenes
   System, „nicht gebucht heißt nicht installiert", klare Abgrenzung. Dagegen: Registry und
   Einrichtungsweg nötig (3–5 PT), und Backup, Restore, Alarme, Runbook vervielfachen sich.
3. **Web oder Desktop** (Lukes Entscheidung 2). Neue Argumente: der Editor ist kein Blocker; die
   fest einkompilierte Serveradresse macht den Installer-Weg bei mehreren Kundenservern teuer;
   „nur gebuchte Module" ist im Web sauberer umsetzbar. Offen: Wie stark wiegt der Eindruck, dass
   ein Programm fertiger wirkt als eine Website?
4. **Registry ja/nein** — bzw. wie Kundenserver an Images kämen, ohne Quellcode zu bekommen.
   Hängt an Punkt 2.
5. **Serveradresse zur Laufzeit** — als einziger Punkt unabhängig von allem anderen sinnvoll.
6. **Website-Fassung und Preise** — läuft ohnehin separat, hängt aber an Punkt 1 (was verkaufen
   wir, und welche Module gibt es zum Start wirklich).
7. **Support-Zusage und Zugriffsweg** für den Piloten — wer außer Luke kommt an SSH-Key und
   Vault-Passwort.

---

## 7 · Nicht geprüft

- Ob die Seitenleiste die Aktivierungstabelle wirklich liest oder heute noch aus den
  Frontend-Konstanten (`business-profiles.ts`) kommt. Für den Einrichtungs-Erzeuger relevant.
- Der Aufwand für die Token-Speicherung im Web wurde nicht beziffert.
- Ob das Weglassen einzelner Frontend-Chunks ohne Nacharbeit funktioniert (die Seitenleiste darf
  ein fehlendes Modul dann nicht mehr anbieten).
- Die G1- und G2-Posten aus dem Lagebild wurden nicht neu bewertet, nur dort markiert, wo „Server
  pro Kunde" sie verteuern würde.
- Der Betriebsaufwand mehrerer Kundenserver ist nicht beziffert — nur benannt. Wer Vorschlag 1
  ernsthaft erwägt, sollte ihn vorher schätzen.
