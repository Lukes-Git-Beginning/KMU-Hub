---
description: Laedt zentria-intel-Keepers fuer ein Cosmi-Modul oder Thema in den Context. Nutzen wenn man an einem Modul arbeitet und Markt-Insights als Hintergrund braucht.
argument-hint: "<modul oder thema> [--days=90]"
---

# intel-recall

Du laedst alle gepickten Intel-Insights ("Keepers") aus `~/Documents/zentria-intel/keepers/` mit passendem Tag in den aktuellen Context, damit der User in einer Modul-Session Markt-Wissen verfuegbar hat.

## Argumente parsen

`$ARGUMENTS` Beispiele:
- `helpdesk` -> alle Keepers mit Tag `modul:helpdesk`
- `crm-core` -> alle Keepers mit Tag `modul:crm-core`
- `ai-in-crm` -> alle Keepers mit Tag `thema:ai-in-crm`
- `helpdesk --days=30` -> nur letzte 30 Tage
- `helpdesk,crm-core` -> Cross-Modul-Recall (komma-separiert)

Default-Window: 90 Tage.

## Workflow

1. **Filter-Set bestimmen:** Parse Argumente. Trenne Module von Themen (Module sind in der 14er-Liste, Themen sind aus `_themes.yaml`).
2. **Files finden:** `grep` durch `~/Documents/zentria-intel/keepers/*.md` nach Frontmatter `modules:` oder `themes:`.
3. **Window-Filter:** Filtere nach `created` >= heute - days.
4. **Sortieren:** Nach `n_sources × trend_score × recency` absteigend.
5. **Synthese:** Wenn >5 Items: schreibe einen kompakten Recall-Block als Markdown-Quote:

   ```markdown
   ## 📥 Intel-Recall: helpdesk (letzte 90 Tage, 8 Items)

   - **Auto-Triage AI ist Pflicht ab 2027** (W19, n_sources=5)
     "Auto-Triage-AI wird zur Tabellenstake, Cosmi-Helpdesk muss bis Sprint 4 mindestens Drafts + Topic-Klassifikation haben."
     Quellen: zendesk.com/blog/..., news.ycombinator.com/...

   - **Zammad 6.5 mit Email-AI-Drafts veroeffentlicht** (W18, n_sources=3)
     ...

   _(weitere {N} Items in keepers/, /intel-recall helpdesk --days=180 fuer mehr)_
   ```

6. **In Context geben:** Als regulaerer Markdown-Output, der dann von Claude in die naechste Antwort eingebettet werden kann.

## Edge-Cases

- **Keine Keepers gefunden:** Zeige `_(noch keine Keepers fuer dieses Modul/Thema)_` und schlage vor `/intel-friday` zu pruefen.
- **>20 Items:** Top-10 zeigen, Rest als `... +{N} weitere`-Hinweis.
- **Followups mit due_at <= heute:** Markiere mit `⏰ FAELLIG` als Praefix.

## Pflege-Tipp im Output

Wenn der User Recall startet ohne dass der Pool >= 5 Items pro Modul hat: Erinnere ihn am Ende, dass das Intel-System noch reift und Keepers wachsen mit jeder Pick-Aktion.
