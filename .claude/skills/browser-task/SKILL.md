---
name: browser-task
description: Führt eine Browser-Aufgabe in einem isolierten Subagenten mit Zeitbudget und Zwischenberichten aus. Nutzen für alles, was Klicken, Formulare, Ablesen oder UI-Prüfung im Browser erfordert — hält Screenshots und DOM-Dumps aus dem Hauptkontext.
argument-hint: "[quick|standard|deep|long] <was zu tun ist>"
context: fork
agent: browser-operator
disable-model-invocation: true
metadata:
  phase: qa
---

# Browser-Auftrag

**Auftrag:** $ARGUMENTS

## Dein Budget

Das erste Wort des Auftrags ist das Preset. Fehlt es, gilt `standard`.

| Preset | Zug-Budget | Checkpoint alle | grobe Wanduhr |
|---|---|---|---|
| `quick` | 15 Züge | 8 | ~10 min |
| `standard` | 30 Züge | 10 | ~20 min |
| `deep` | 60 Züge | 15 | ~40 min |
| `long` | 120 Züge | 20 | ~75 min |

Ein **Zug** ist eine deiner Antworten, unabhängig davon wie viele Werkzeugaufrufe darin stecken —
ein `browser_batch` mit fünf Aktionen ist ein Zug, nicht fünf. Zähle mit. Die Wanduhr-Spalte ist
Erfahrungswert, keine Garantie — maßgeblich sind die Züge.

Bist du **vor dem ersten Checkpoint fertig**, entfällt er — der Schlussbericht genügt.
Checkpoints sichern lange Läufe ab, sie sind kein Ritual.

**Bei jedem Checkpoint** meldest du dich per `SendMessage` an `main`: Stand, Rest-Budget, Blocker
oder Entscheidungsfrage, Einschätzung ob das Budget reicht. Danach arbeitest du weiter, es sei denn,
eine Antwort weist dich um. Antwortet niemand, ist das kein Grund zu stoppen — der Zwischenstand ist
gesichert, das genügt.

**Budget aufgebraucht:** Schlussbericht mit dem Erreichten, klarer Liste des Fehlenden und der
Einschätzung, welches Preset für den Rest nötig wäre. Nicht heimlich überziehen.

## Ablauf

1. **Werkzeuge in einem einzigen `ToolSearch`-Aufruf laden** (Liste in deinem Systemprompt).
2. **Verbinden:** `tabs_context_mcp`, um die offenen Tabs zu sehen. Neuen Tab per `tabs_create_mcp`
   öffnen, statt einen bestehenden zu kapern — es sei denn, der Auftrag nennt ausdrücklich einen.
3. **DOM oder Canvas feststellen** (`get_page_text`) — das entscheidet, ob du über Selektoren oder
   über Vision arbeitest.
4. **Auftrag ausführen**, nach Bereichen gebündelt, mit den Vorsichtsregeln aus deinem Systemprompt.
5. **Schlussbericht** im Vier-Abschnitte-Format: Ergebnis · Nicht erfasst · Auffälligkeiten ·
   Unsicherheiten.

## Wenn der Browser nicht mitspielt

- **„Browser extension is not connected" / „Receiving end does not exist"** — der Service-Worker der
  Extension ist eingeschlafen. Das kannst du **nicht selbst reparieren**: `/chrome` läuft nur in der
  Hauptsession. Melde es per `SendMessage` an `main` mit der Bitte um `/chrome` → „Reconnect
  extension" und beende den Durchgang, statt zu pokern.
- **Seite reagiert nicht / Blackscreen** — einmal neu navigieren. Hilft das nicht: melden.
- **Login oder CAPTCHA** — nicht umgehen. Melden und abbrechen.
- **Nach 2–3 fehlgeschlagenen Versuchen derselben Aktion**: aufhören, melden. Wiederholung derselben
  fehlschlagenden Aktion verbrennt nur Budget.
