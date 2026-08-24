#!/usr/bin/env python3
"""Bilanz fuer einen Backend-Nachtlauf (run-loop.ps1 ruft dies am Laufende auf).

Liest JOURNAL.md, zaehlt die Iterationen eines Laufbereichs nach Status/Praefix/
Coverage-Delta/offenen Entscheidungen aus und haengt einen zusammenfassenden
Markdown-Block ans Dateiende an. Ohne diese Zahl musste die letzte Merge-Sitzung
die Substanz eines Laufs aus den Commit-Praefixen von Hand rekonstruieren.

Der Treiber ruft auf:
    python run-summary.py --base-sha <sha> --from <int> --to <int> --minutes <int> [--dry-run]

Das Journal haelt sich nicht strikt an sein eigenes Format (ITERATION.md:253-263) --
Trennzeichen wechseln zwischen Gedankenstrich und Bindestrich, Statusangaben und
Coverage-Schreibweisen variieren. Was sich nicht sicher parsen laesst, wird als
"nicht auswertbar" gezaehlt statt stillschweigend verschluckt (siehe iterationen:-Zeile).
"""
import argparse
import re
import subprocess
import sys
from collections import Counter
from datetime import datetime
from pathlib import Path

# Pfade relativ zum Skript aufloesen, nicht relativ zum Arbeitsverzeichnis --
# der Treiber kann aus dem Repo-Root oder aus dem Loop-Verzeichnis heraus rufen.
LOOP_DIR = Path(__file__).resolve().parent.parent
REPO_ROOT = (
    LOOP_DIR.parent.parent.parent
)  # loop -> backend-block -> .planning -> Repo-Root
JOURNAL_PATH = LOOP_DIR / "JOURNAL.md"

# "## Iteration 93 - unit-id - done - 2026-08-23 10:32" oder mit "—" statt "-".
# Die Nummer wird separat und robust extrahiert (siehe NUM_RE) -- nur ueber sie
# laesst sich ein Eintrag ueberhaupt einem Laufbereich zuordnen. Der Rest hier
# wird tolerant weiterzerlegt und darf scheitern, ohne die Nummer zu gefaehrden.
NUM_RE = re.compile(r"^##\s*Iteration\s+(\d+)")
ITERATION_HEADER_RE = re.compile(r"^##\s*Iteration\s+\d+\s*[—–-]\s*(.*)$")
# Trennt Unit-ID / Status / Zeitstempel. Braucht Leerraum auf beiden Seiten des
# Bindestrichs, damit Bindestriche INNERHALB einer Unit-ID (fix-idempotency-409-...)
# nicht als Trenner missverstanden werden.
FIELD_SPLIT_RE = re.compile(r"\s+[—–-]\s+")
# Eine Pflichtzeile wie "- coverage: ..." oder "- mutations-probe: ...".
BULLET_RE = re.compile(r"^-\s*([\w -]+?):\s*(.*)$")
# "keine"/"keine."/"n.a."/"n.a"/"-" als alleinstehendes erstes Wort markiert eine
# leere offen:-Zeile, auch wenn danach noch Begruendungstext folgt.
EMPTY_MARKER_RE = re.compile(r"^(keine\.?|n\.a\.?|-)(\s|$)", re.IGNORECASE)
# "<paket> <vorher> % ... -> ... <nachher> %", tolerant gegen Backticks/Bold/
# Klammer-Kommentare dazwischen (werden vorher entfernt bzw. ueberlesen).
COVERAGE_ITEM_RE = re.compile(
    r"(?P<pkg>[A-Za-z][\w./]*)\s+(?P<before>\d+(?:[.,]\d+)?)\s*%.*?"
    r"->\s*.*?(?P<after>\d+(?:[.,]\d+)?)\s*%"
)


def parse_args():
    p = argparse.ArgumentParser(description="Bilanz eines Backend-Nachtlaufs erzeugen")
    p.add_argument(
        "--base-sha", required=True, help="Commit, auf dem der Lauf startete"
    )
    p.add_argument("--from", dest="from_iter", type=int, required=True)
    p.add_argument("--to", dest="to_iter", type=int, required=True)
    p.add_argument(
        "--minutes", type=int, required=True, help="Gesamtlaufzeit in Minuten"
    )
    p.add_argument(
        "--dry-run",
        action="store_true",
        help="Block auf stdout statt an JOURNAL.md anhaengen",
    )
    return p.parse_args()


def strip_markdown(text):
    """Entfernt Backticks/Fett-Markierungen, die in Journal-Zeilen um Paketnamen
    oder Zahlen stehen (`internal/gateway` **56,6 % -> 56,6 %**)."""
    return text.replace("`", "").replace("**", "")


def collect_blocks(lines):
    """Zerlegt den Journal-Text in (Header-Zeile, Restzeilen)-Bloecke, jeweils ab
    einer '## Iteration'-Zeile bis zur naechsten (oder Dateiende). '## CI nach
    Lauf (...)'-Bloecke und alles davor faellt automatisch heraus, weil dort
    keine '## Iteration'-Zeile beginnt."""
    starts = [i for i, l in enumerate(lines) if l.startswith("## Iteration")]
    starts.append(len(lines))
    return [lines[starts[i] : starts[i + 1]] for i in range(len(starts) - 1)]


def extract_bullet(block, name):
    """Sammelt den vollstaendigen Text einer Pflichtzeile (z. B. 'coverage'),
    inklusive eingerueckter Folgezeilen und Unterbullets, bis die naechste
    '- xyz:'-Zeile, eine Leerzeile oder '---' den Block beendet."""
    out = []
    capturing = False
    for line in block[1:]:
        m = BULLET_RE.match(line)
        if m:
            if capturing:
                break
            if m.group(1).strip().lower() == name:
                capturing = True
                out.append(m.group(2))
            continue
        if capturing:
            stripped = line.strip()
            if stripped in ("", "---"):
                break
            # Markdown-Unterbullet-Marker der Folgezeile entfernen (offen: kommt
            # oft als "- offen:\n  - Punkt eins\n  - Punkt zwei") -- sonst
            # ueberlebt ein fuehrender "-" den Join und taeuscht EMPTY_MARKER_RE
            # eine leere Zeile vor.
            if stripped.startswith("- "):
                stripped = stripped[2:]
            elif stripped == "-":
                stripped = ""
            out.append(stripped)
    return " ".join(out).strip()


def parse_iteration(block):
    """Liefert dict(num, unit, status) oder None, wenn die Kopfzeile keine
    Iterationsnummer traegt (dann ist der Block keinem Laufbereich zuordenbar
    und faellt komplett aus der Auswertung, statt ein fremdes '## Iteration'-
    Vorkommen faelschlich mitzuzaehlen). status ist None, wenn er sich trotz
    vorhandener Nummer nicht sicher bestimmen liess -- das zaehlt im
    Laufbereich als 'nicht auswertbare Kopfzeile', nicht als Fehler."""
    m_num = NUM_RE.match(block[0])
    if not m_num:
        return None
    num = int(m_num.group(1))
    unit, status = "unbekannt", None
    m_rest = ITERATION_HEADER_RE.match(block[0])
    if m_rest:
        parts = FIELD_SPLIT_RE.split(m_rest.group(1))
        unit = parts[0].strip() if parts and parts[0].strip() else "unbekannt"
        if len(parts) >= 2 and parts[1].strip().lower() in ("done", "blocked"):
            status = parts[1].strip().lower()
    return {"num": num, "unit": unit, "status": status, "block": block}


def coverage_deltas(entries):
    """Erster Vorher-Wert und letzter Nachher-Wert je Paket ueber den Laufbereich
    (nicht aufsummieren -- dieselben Pakete werden mehrfach gemessen). Reihenfolge
    der Rueckgabe: Erscheinungsreihenfolge im Journal (fuer stabile Ausgabe)."""
    first_before = {}
    last_after = {}
    order = []
    for e in entries:
        text = strip_markdown(extract_bullet(e["block"], "coverage"))
        if not text or text.lower().startswith("n.a"):
            continue
        for segment in re.split(r"\s*·\s*", text):
            m = COVERAGE_ITEM_RE.search(segment)
            if not m:
                continue
            pkg = m.group("pkg")
            if pkg not in first_before:
                first_before[pkg] = m.group("before")
                order.append(pkg)
            last_after[pkg] = m.group("after")
    # lean: Paketname wird woertlich aus dem Journal uebernommen, keine
    # Kanonisierung von Kurzformen ("quote" vs. "internal/biz/quote"). Faellt
    # erst dann auf, wenn dieselbe Kurzform in verschiedenen Iterationen einen
    # ECHTEN Delta traegt -- dann auf eine Alias-Tabelle je Paketpfad upgraden.
    changed = [
        (pkg, first_before[pkg], last_after[pkg])
        for pkg in order
        if first_before[pkg] != last_after[pkg]
    ]
    changed.sort(key=lambda t: abs(_to_float(t[2]) - _to_float(t[1])), reverse=True)
    return changed


def _to_float(value):
    try:
        return float(value.replace(",", "."))
    except ValueError:
        return 0.0


def offen_stats(entries):
    """Zaehlt nicht-leere offen:-Zeilen und darunter die mit Treffer auf
    "luke"/"entscheidung" (case-insensitiv, wortwoertliche Substring-Suche)."""
    non_empty = 0
    needs_decision = 0
    for e in entries:
        text = extract_bullet(e["block"], "offen")
        if not text or EMPTY_MARKER_RE.match(text):
            continue
        non_empty += 1
        low = text.lower()
        if "luke" in low or "entscheidung" in low:
            needs_decision += 1
    return needs_decision, non_empty


def commit_type(subject):
    """Conventional-Commit-Praefix vor der ersten '(' oder ':'. Ohne eine von
    beiden (Freitext-Betreff) landet der Commit unter 'sonstige'."""
    idx_candidates = [i for i in (subject.find("("), subject.find(":")) if i != -1]
    if not idx_candidates:
        return "sonstige"
    prefix = subject[: min(idx_candidates)].strip()
    return prefix.lower() if prefix and prefix.isalpha() else "sonstige"


def commits_by_type(base_sha):
    """Zaehlt Commits seit base_sha nach Conventional-Commit-Typ. Liefert
    (counter, gesamt, fehlermeldung) -- fehlermeldung ist None bei Erfolg.
    Ein git-Fehler darf die Bilanz nie zum Absturz bringen."""
    try:
        result = subprocess.run(
            ["git", "log", f"{base_sha}..HEAD", "--format=%s"],
            cwd=REPO_ROOT,
            capture_output=True,
            text=True,
            encoding="utf-8",
        )
    except OSError as exc:
        return None, 0, str(exc)
    if result.returncode != 0:
        grund = (
            result.stderr.strip().splitlines()[-1]
            if result.stderr.strip()
            else f"exit {result.returncode}"
        )
        return None, 0, grund
    subjects = [s for s in result.stdout.splitlines() if s.strip()]
    counter = Counter(commit_type(s) for s in subjects)
    return counter, len(subjects), None


def format_counter(counter):
    return " · ".join(f"{k} {v}" for k, v in counter.most_common())


def fmt_num(value):
    """Deutsche Kommaschreibweise fuer eine bereits als String vorliegende Zahl."""
    return value.replace(".", ",")


def build_block(args, entries):
    done = sum(1 for e in entries if e["status"] == "done")
    blocked = sum(1 for e in entries if e["status"] == "blocked")
    kein_status = len(entries) - done - blocked

    prefix_counter = Counter(e["unit"].split("-")[0] for e in entries)

    counter, commit_total, grund = commits_by_type(args.base_sha)
    if grund is not None:
        commits_line = f"- commits nach typ: nicht ermittelbar ({grund})"
    else:
        base_kurz = args.base_sha[:8]
        commits_line = f"- commits nach typ: {format_counter(counter)} ({commit_total} seit {base_kurz})"

    changed = coverage_deltas(entries)
    if not changed:
        coverage_line = "- coverage-delta: keine Aenderungen erkannt"
    else:
        shown = " · ".join(
            f"{pkg} {fmt_num(before)} -> {fmt_num(after)}"
            for pkg, before, after in changed[:3]
        )
        rest = len(changed) - 3
        coverage_line = f"- coverage-delta: {shown}" + (
            f" · ({rest} weitere)" if rest > 0 else ""
        )

    needs_decision, non_empty = offen_stats(entries)

    n = len(entries)
    minutes_per_iter = fmt_num(f"{args.minutes / n:.1f}") if n else "n.a."

    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M")
    lines = [
        f"## Bilanz Lauf ({timestamp})",
        f"- iterationen: {n} im Bereich {args.from_iter}-{args.to_iter}, davon {done} done, "
        f"{blocked} blocked, {kein_status} ohne auswertbare Kopfzeile",
        f"- units nach praefix: {format_counter(prefix_counter) or 'keine Units im Bereich'}",
        commits_line,
        coverage_line,
        f"- offen mit entscheidungsbedarf: {needs_decision} von {non_empty} nicht-leeren offen:-Zeilen "
        f'(Treffer auf "Luke"/"Entscheidung")',
        f"- minuten je iteration: {minutes_per_iter} ({args.minutes} gesamt)",
    ]
    return "\n".join(lines)


def main():
    # Windows/Git-Bash setzt sys.stdout oft auf die Konsolen-Codepage (cp1252)
    # statt UTF-8 -- ohne das hier geht "·"/Umlaute im --dry-run-Druck kaputt.
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            stream.reconfigure(encoding="utf-8")

    args = parse_args()
    try:
        text = JOURNAL_PATH.read_text(encoding="utf-8")
    except OSError as exc:
        print(f"Fehler: {JOURNAL_PATH} nicht lesbar: {exc}", file=sys.stderr)
        sys.exit(1)

    blocks = collect_blocks(text.split("\n"))
    parsed = [parse_iteration(b) for b in blocks]
    entries = [
        p
        for p in parsed
        if p is not None and args.from_iter <= p["num"] <= args.to_iter
    ]

    block = build_block(args, entries)

    if args.dry_run:
        print(block)
        sys.exit(0)

    if not text.endswith("\n"):
        text += "\n"
    JOURNAL_PATH.write_text(text + "\n" + block + "\n", encoding="utf-8")
    print(
        f"run-summary: Bilanz an {JOURNAL_PATH.name} angehaengt ({len(entries)} Iterationen ausgewertet)."
    )
    sys.exit(0)


if __name__ == "__main__":
    main()
