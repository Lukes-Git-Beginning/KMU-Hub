# GSD: Plan Phase

Du erstellst **Execution Plans** fuer die aktuelle Phase basierend auf dem CONTEXT-File.

## Voraussetzung

- CONTEXT-File muss existieren: `.planning/phases/{PHASE_NAME}/{PHASE}-CONTEXT.md`
- Falls nicht vorhanden: Erst `/gsd-discuss-phase` ausfuehren

## Deine Aufgabe

1. **Lies das CONTEXT-File** und verstehe Scope, Decisions, Specifics

2. **Recherchiere den Codebase** fuer jeden geplanten Plan:
   - Welche Dateien muessen erstellt/modifiziert werden?
   - Welche bestehenden Patterns/Services werden wiederverwendet?
   - Welche Abhaengigkeiten zwischen Plans bestehen?

3. **Teile die Phase in Plans auf:**
   - Jeder Plan ist ein eigenstaendiger, commitbarer Arbeitsschritt
   - Typisch: 2-5 Plans pro Phase
   - Reihenfolge: Data Foundation → Backend Services → gRPC+Gateway → Frontend
   - Plans koennen aufeinander aufbauen (depends_on)

4. **Erstelle Plan-Files** im Format:

```markdown
---
phase: {phase-slug}
plan: {NN}
type: execute
wave: {N}
depends_on: [{vorherige plan nummern}]
files_modified:
  - {dateipfad}
autonomous: true
requirements:
  - {REQ-ID}

must_haves:
  truths:
    - "{Aussage die nach Completion wahr sein muss}"
  artifacts:
    - path: "{dateipfad}"
      provides: "{was die Datei bereitstellt}"
      contains: "{string der in der Datei vorkommen muss}"
  key_links:
    - from: "{datei A}"
      to: "{datei B}"
      via: "{wie sie zusammenhaengen}"
      pattern: "{regex pattern}"
---

<objective>
{Ein klarer Absatz der das Ziel beschreibt}
</objective>

<context>
{Referenzen zu CONTEXT-File und vorherigen Plans}
</context>

<tasks>
<task type="auto">
  <name>{Task-Name}</name>
  <files>{Dateien die erstellt/modifiziert werden}</files>
  <action>{Detaillierte Implementierungs-Anweisungen}</action>
  <verify>{Verifikations-Schritte}</verify>
  <done>{Completion-Statement}</done>
</task>
</tasks>

<verification>
{Finale Verifikations-Checkliste}
</verification>

<success_criteria>
{Bash-Kommandos oder Bedingungen die Erfolg beweisen}
</success_criteria>

<output>
Create a SUMMARY file after completion using the summary format.
</output>
```

## Plan-Aufteilung Patterns (aus 90 Plans gelernt)

- **Data Foundation:** Migration + Proto + Go Models + Repository (1 Plan)
- **Backend Services:** Service Layer + Business Logic (1-2 Plans)
- **gRPC + Gateway:** Server Registration + HTTP Routes (1 Plan)
- **Frontend:** Types + Hooks + Components + Pages (1-2 Plans)

## Wichtig

- Jeder Plan muss eigenstaendig ausfuehrbar und verifizierbar sein
- `must_haves` sind die harten Erfolgskriterien — keine weichen Formulierungen
- `artifacts` mit `contains` ermoeglichen automatische Verifikation
- Plans IMMER in `.planning/phases/{PHASE_NAME}/` ablegen
