# Backend-Status Cosmi — Stand 20.06.2026

*Kurzüberblick für Darien · alle Zahlen direkt aus dem Code verifiziert*

**In einem Satz:** Das Backend ist im Kern fertig — alle 23 Fachdienste laufen, 17 davon
vollständig; die übrigen 5 haben nur bewusst zurückgestellte Zusatzfeatures (keine
Kernblocker). Offen ist die finale Sicherheits- und Stabilitätshärtung (Rigorosum-Runde 3:
4 von 9 kritischen Punkten erledigt).

```
Services voll fertig     ███████████████░░░░░   17 / 23   (74 %)
Services lauffähig        ████████████████████   23 / 23   (100 %, 0 leere Stubs)
Sicherheit R3 (P0)        █████████░░░░░░░░░░░    4 / 9    erledigt
Mandantentrennung aktiv   █████████░░░░░░░░░░░   ~10 / 23  Dienste (Fundament: 178+ Tab. ✓)
Consent Anruf + E-Mail    ████████████████░░░░   beide Pfade verdrahtet
Module live geschaltet    ░░░░░░░░░░░░░░░░░░░░    0 / 14   (gebaut, bewusst per Flag AUS)
Test-Gate (CI)            ✓ erfüllt              15 % Pflicht · aktuell ~20 %
Reifegrad-Note            3,0                    Ziel Pilot-Start: 2,3  (1 = top)
```

## Was steht

- 24 Bausteine (1 Gateway + 23 Fachdienste), alle gestartet und zentral verdrahtet.
- 17 Dienste komplett: CRM, Dialer, Chat, E-Mail, Dokumente, Kalender/Aufgaben/Video,
  Schichten, Helpdesk, Wiki, Formulare, Berichte, Einkauf, Produktion, Fuhrpark, Inventar,
  Benachrichtigungen, Plugins.
- Datenbank: 187 Migrationen (Kopf 218). Mandantentrennung (RLS) auf 178+ Tabellen,
  zusätzlich pro DB-Verbindung abgesichert.
- Rechtssicherheit: Einwilligungsprüfung bei Anrufen **und** E-Mail-Versand aktiv.
- Qualität: CI prüft bei jedem Push Code-Stil, Tests (inkl. Race-Detector), End-to-End und
  API-Spezifikation; Finanz-, CRM-, Dialer- und Auth-Kern sind solide getestet.

## Wo aktuell noch Arbeit liegt

*(Momentaufnahme, keine Terminplanung)*

- 5 Dienste mit zurückgestellten Zusatzfeatures, kein Kernblocker: biz (Lexware-Sync,
  GoBD-CSV-Export), crm (CSV/vCard-Import), rapporte & vertraege (PDF noch als Text),
  vermietung (Foto-Upload Platzhalter).
- 5 von 9 Sicherheits-/Stabilitätspunkten aus Rigorosum-Runde 3 offen (Schwerpunkt:
  Mandantentrennung in den restlichen Diensten scharf schalten, Deploy-Konfiguration,
  Recording-Consent-Anzeige).
- Module sind fertig gebaut, aber per Feature-Flag deaktiviert — werden je Pilot
  kontrolliert freigeschaltet.

## Einordnung

Reifegrad-Note aktuell **3,0** (Vorrunden 3,3 → 4,1 → 3,0, also wieder aufwärts).
Für den Start des ersten Piloten ist eine 2,3 angepeilt.
