# Orbit Pod — Custom-Appliance Teileliste & Konzept

> **Stand:** 2026-06-09 · Arbeitsstand aus dem Gespräch mit Darien. Referenz fürs Team + den ORBIT-Wizard.
> **Ziel:** Eigene, kompakte 24/7-Appliance für **Kleinunternehmer (2-10 MA)** in **Custom-Hülle mit Front-Display** ("Orbit Console") — gibt dem Kunden das Gefühl von Kontrolle, läuft die komplette Cosmi/Orbit-Software lokal. Tier = **Pod** (offiziell 5-20 User; 2-10 sitzt bequem im unteren Bereich).
> **Status:** Einkaufs-/Designgrundlage, **kein final validiertes BOM**. Exakte SKUs + Tagespreise beim Beschaffen gegenchecken. Thermik-Burn-in + CE/EMV stehen vor Verkauf aus.

## Build-Philosophie
Lüfterloses bzw. leises **Mini-ITX-Gerät**, Linux + Docker, kompletter Cosmi-Stack lokal (Gateway + Microservices + PostgreSQL 16 + Redis). **7"-Touchdisplay** in der Front zeigt eine Kiosk-"Orbit-Console" (Service-Health, Backup-Status, Speicher, User online, Updates) — der eigentliche Differenzierer ist diese Display-Software (baubar aus dem vorhandenen Prometheus/Grafana-Monitoring + schlanker Kiosk-PWA), nicht die rohe Hardware.

---

## A) Consumer-Prototyp (Lean, fanless, ohne lokales Video)
Schnell baubar, billig, zum Konzept beweisen.

| Teil | Beispiel (o. ä.) | Maße | ~€ |
|---|---|---|---|
| Board+CPU | ASRock **N100DC-ITX** (N100, DC-in onboard, fanless) | 170 × 170 mm | 140 |
| RAM | Crucial 16 GB DDR4-3200 SO-DIMM | 30 × 67 mm | 35 |
| System-SSD | WD Blue SN580 / Crucial P3 512 GB NVMe | M.2 2280 (22 × 80) | 40 |
| Daten + Backup | 2× Crucial MX500 2 TB 2,5″ (RAID 1) | 100 × 70 × 7 mm je | 240 |
| Netzteil | 12 V / 120 W Brick (an DC-in) | extern | 25 |
| Kühlung | Passiv-Kühlkörper | flach | 0-30 |
| Display | Waveshare 7″ HDMI IPS Touch 1024×600 | ~165 × 105 × 15 mm | 70 |
| Gehäuse | generisches Mini-ITX-Case / Test-Frame | — | 50 |
| Kleinteile | Kabel, Button, LED, Standoffs | — | 25 |
| **Summe** | | | **~625 €** |

**→ Video-Upgrade (Consumer):** Board+CPU → Mini-ITX B760 + **i5-13500T** (~400), RAM 32 GB DIMM (~80), **Noctua NH-L9i** Low-Profile-Kühler (~45), **Corsair SF450** SFX-Netzteil (~85), größeres Gehäuse. **Delta ~+450 € → ~1.050 €.**

---

## B) Industrial/ECC-Produktion (verkaufsreif, 24/7, ECC + Fernwartung)
Für das verkaufte Produkt empfohlen.

| Teil | Beispiel (o. ä.) | Maße | ~€ |
|---|---|---|---|
| Board+CPU | Supermicro **A2SDi-4C-HLN4F** (Atom C3558, **ECC**, **IPMI onboard**, fanless) | 170 × 170 mm | 380 |
| RAM | Kingston/Micron 32 GB DDR4 **ECC** SO-DIMM | 30 × 67 mm | 95 |
| System-SSD | Samsung **PM893** 480 GB / WD Red SN700 (hohe Endurance) | M.2 / 2,5″ | 75 |
| Daten + Backup | 2× WD Red SA500 2 TB / Samsung PM893 (RAID 1, hohe TBW) | 100 × 70 × 7 mm je | 360 |
| Netzteil | Qualitäts-12V-Industrie-DC (opt. redundant) | extern | 60 |
| Fernwartung | **IPMI/BMC onboard** | — | inkl. |
| Display | Industrie-7″-Touch (weiter Temp-Bereich) | ~165 × 105 mm | 90 |
| Gehäuse | Fanless-Industrie-Chassis / Custom-Alu | — | 100-250 |
| Kleinteile | Kabel, Button, LED, Standoffs | — | 30 |
| **Summe** | | | **~1.200-1.350 €** |

**→ Video-Upgrade (Industrial):** Atom C3558 ist für Video zu schwach → **Xeon-D-Board** (Supermicro **X12SDV**, ECC+IPMI, ~600-900 Board+CPU) **oder** ASRock Rack **Ryzen-mini-ITX** (Ryzen 5/7 mit ECC + BMC) + aktive Kühlung + stärkeres Netzteil. **Delta ~+400-600 €.**

---

## Gehäuse (Custom-Hülle) — Dimensionen
Internes Mindestvolumen: Mini-ITX-Board (170×170) + Kühler-Höhe + 2-3 SSDs + Netzteil + Display + Verkabelung.
- **Liegend (Würfel):** ca. **210 × 200 × 110 mm** (Lean) bzw. **240 × 220 × 150 mm** (Video, aktive Kühlung).
- **Stehend (Tower / "Statussäule"):** ca. **160 × 170 × 230 mm** (Lean) bzw. **180 × 200 × 280 mm** (Video).
- **Lüftung:** umlaufende Passiv-Schlitze oben+unten (Kamineffekt bei fanless); bei Video mehr Lüftungsfläche.
- **Material/Finish:** Alu-Front (zugleich Heatsink-Fläche bei CPU-an-Wand) + ggf. Holz/Filz-Akzent (passt zur Cosmi-Editorial-Ästhetik). Front: 7"-Display + Power-Button + dezente Status-LED + Orbit/Zentria-Logo. Keine Gamer-Optik.

### Component-Footprints (fürs CAD)
| Teil | Footprint |
|---|---|
| Mini-ITX-Board | 170 × 170 mm |
| Passiv-/Low-Profile-Kühler | 40-60 mm Höhe |
| 2,5″ SSD | 100 × 69,85 × 7 mm |
| (3,5″ HDD — eher meiden) | 147 × 101,6 × 26,1 mm |
| M.2 2280 NVMe | 22 × 80 mm |
| 7″ Display-Modul | ~165 × 105 mm Sichtfläche, ~15 mm tief |
| SFX-Netzteil (Video) | 125 × 63 × 100 mm |

---

## Video — Optionen & der echte Engpass
- **Cloud (Standard):** Meeting-Server auf Hetzner-VPS, im Modulpreis enthalten (bis 10 Teiln.) — keine Extra-Hardware.
- **Lokal ("in Orbit"):** SFU (LiveKit/coturn) auf der Video-Variante der Box bzw. separatem Knoten.
- **Hybrid (empfohlen):** intern lokal, extern Cloud.
- **Engpass = Upload-Bandbreite**, nicht CPU. LAN-/interne Calls problemlos; **externe** Teilnehmer limitiert der Firmen-Upload (typ. DSL bricht bei wenigen externen Streams ein). Ironie: Orbit-Zielgruppe "ländlich/schlechtes Internet" ist für self-hosted *Außen*-Video am schlechtesten dran → extern standardmäßig Cloud.
- **Netzwerk/Ports:** coturn **3478/5349** + LiveKit-UDP-Portrange erreichbar; bei externem Video Port-Forwarding/feste-ish IP nötig. **Dual-NIC** empfohlen (LAN voll / WAN kontrolliert).

## GPU? — Nein
- SFU **leitet weiter, encodiert nicht** → kein GPU. Server-Encoding nur bei **Aufzeichnung/Composite** → deckt die **integrierte GPU (Intel QuickSync)** ab. **Intel ohne "F"** wählen (F = keine iGPU); bei Ryzen ein Modell **mit** Radeon-iGPU.
- Diskrete GPU nur bei **lokaler KI** (Transkription/Assistent on-device) — andere Hardware-Klasse.

## KI — Stand
KI ist **erstmal nur Cloud auf eigenem Hetzner** geplant (selbst gebaut), erst **nach** Cosmi. → Orbit-Box braucht dafür **nichts** außer Internet-Zugang zum KI-Endpoint (API-Call). **Der GPU-Bedarf wandert in den künftigen Hetzner-KI-Server** (eigenes Projekt, VRAM-getrieben), getrennt von den Kunden-Appliances. Datenschutz-Notiz: KI-in-Cloud heißt, Daten verlassen für KI-Features die lokale Box (auch wenn EU-Hetzner) — für strengste Kunden später evtl. optionaler **lokaler KI-Knoten** (GPU-Variante) als Premium-Add-on.

---

## Consumer vs. Industrial — was zählt für ein verkauftes Gerät
- **ECC-RAM:** fängt Speicher-Bitflips → Datenintegrität (Buchhaltung/CRM!). "Bit-rot-sicher" = Verkaufsargument. Braucht CPU+Board mit ECC (die meisten Consumer-Intel nicht; viele Ryzen + Embedded-Boards ja).
- **Fernverwaltung (IPMI/BMC oder KVM-over-IP):** remote rebooten/neu aufsetzen ohne hinfahren = euer Wartungs-Geschäftsmodell. Industrial-Boards: onboard. Consumer: extern PiKVM (~80) + Smart-PDU/Relais (~40) nachrüsten.
- **Redundante NT / Hot-Swap / Rack / Xeon-EPYC:** Overkill für 2-10 → weglassen.
- **Empfehlung:** Prototyp = Consumer (N100/i5). Verkauftes Produkt = **Industrial/Embedded SFF mit ECC + Fernzugriff-Pfad**.

## Kosten-Überblick (Teile, grob)
| Konfig | ~€ Teile |
|---|---|
| Consumer-Lean | ~625 |
| Consumer-Video | ~1.050 |
| Industrial-Lean (ECC+IPMI) | ~1.200-1.350 |
| Industrial-Video (Xeon-D/Ryzen-ECC) | ~1.700-2.000 |

> Vergleich: Synology-Pod von der Stange ~900 € (DS423+) — die Custom-Appliance liegt ähnlich (Lean) bis darüber (Industrial/Video), liefert aber Display + volle Design-Kontrolle + ECC/Fernwartung.

## Pricing-Kontext (aus `.knowledge/pricing.md`)
- Pod-Tier: Hardware ~900 € Kauf / ~30 €/Mo Lease · Setup 199 € (remote) · Wartung 39 €/Mo.
- Cloud-Backup-Paket **S (bis 1 TB) = 9 €/Mo** (für 2-10 dick ausreichend).
- Orbit-Rabatt: -20 % auf alle Cosmi-Modulpreise.

## Offene Punkte / To-dos
- [ ] Exakte ECC-fähige SKU + Tagespreise verifizieren (Supermicro A2SDi / ASRock Rack Ryzen / Xeon-D).
- [ ] Thermik-Burn-in je Variante (besonders fanless + Video-aktiv).
- [ ] CE/EMV/Niederspannung klären (externes zertifiziertes Netzteil entschärft viel).
- [ ] **Doku-Inkonsistenz glätten:** `pricing.md` nennt **Jitsi** als Meeting-Server, das Produkt nutzt **LiveKit** (+ coturn) — Außendarstellung auf eins festlegen (mit Luke).
- [ ] "Orbit-Console" Kiosk-UI fürs Display spezifizieren (aus Prometheus/Grafana + PWA).
- [ ] Custom-Hülle: Maßskizze (Tower mit 7"-Front) + Fertigungsweg (CNC-Alu vs. gekantet/3D-Druck für Proto).
