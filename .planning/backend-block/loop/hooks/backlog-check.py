#!/usr/bin/env python3
"""Backlog-Check fuer den Nachtloop-Treiber (run-loop.ps1).

Ersetzt die zwei fragilen Zeilen-Regex-Funktionen `Get-NextUnitModel` und
`Get-OpenUnitCount` im Treiber (run-loop.ps1:306-324) durch echtes YAML-Parsing
ueber die drei Backlog-Dateien im selben Verzeichnis.

Modi:
    --preflight   Validiert BACKLOG.yml, BACKLOG-NEXT.yml, BACKLOG-PARKED.yml.
                  Meldet ALLE gefundenen Verstoesse gesammelt auf stderr,
                  Exit 1 bei mindestens einem Verstoss, sonst eine Erfolgszeile
                  auf stdout und Exit 0.
    --state       Liest NUR BACKLOG.yml und schreibt genau drei Zeilen
                  (OPEN=, NEXT=, MODEL=) auf stdout. Exit 0, ausser bei
                  YAML-Fehler (Meldung auf stderr, Exit 1).
    --model-of ID Schreibt eine Zeile MODEL=<sonnet|opus> fuer genau diese Unit,
                  gesucht ueber alle drei Dateien. Unbekannte ID -> MODEL= (leer).
                  Immer Exit 0.
"""
import re
import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("Fehler: PyYAML fehlt. Installieren mit: pip install pyyaml", file=sys.stderr)
    sys.exit(1)

# Pfade relativ zum Skript-Verzeichnis aufloesen (nicht relativ zum Arbeitsverzeichnis --
# der Treiber ruft aus dem Repo-Root auf, andere aus dem Loop-Verzeichnis).
LOOP_DIR = Path(__file__).resolve().parent.parent
BACKLOG = LOOP_DIR / "BACKLOG.yml"
BACKLOG_NEXT = LOOP_DIR / "BACKLOG-NEXT.yml"
BACKLOG_PARKED = LOOP_DIR / "BACKLOG-PARKED.yml"

# done_when[0], das auf eine Entscheidung von Luke zeigt -- Umlaut- UND ASCII-Schreibweise
# kommen beide in den Dateien vor.
LUKE_DECISION_RE = re.compile(
    r"luke hat|entscheidung geh(?:oe|ö)rt luke", re.IGNORECASE
)

VALID_MODELS = ("sonnet", "opus")
OPEN_STATUSES = ("todo", "in_progress")


def load_yaml(path):
    """Laedt eine Backlog-Datei. Gibt (data, fehlermeldung) zurueck --
    fehlermeldung ist None bei Erfolg, sonst ein fertiger String mit Datei/Zeile/Spalte."""
    try:
        text = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return None, f"{path.name}: Datei nicht gefunden"
    try:
        return yaml.safe_load(text), None
    except yaml.YAMLError as e:
        mark = getattr(e, "problem_mark", None)
        ort = (
            f"Zeile {mark.line + 1}, Spalte {mark.column + 1}"
            if mark
            else "Position unbekannt"
        )
        problem = getattr(e, "problem", None) or str(e)
        return None, f"{path.name}: YAML-Fehler bei {ort}: {problem}"


def units_of(data):
    """Extrahiert die units-Liste aus dem geparsten Dokument, oder None wenn die
    Struktur (Top-Level-Key 'units' mit einer Liste) nicht passt."""
    if not isinstance(data, dict):
        return None
    units = data.get("units")
    return units if isinstance(units, list) else None


def find_cycle(units):
    """Einfache DFS-Zyklensuche ueber die deps EINER Unit-Liste (nur unit-lokale
    IDs werden aufgeloest, fehlende deps sind hier kein Fall -- die meldet der
    Aufrufer bereits separat). Gibt die ID-Kette des ersten gefundenen Zyklus
    zurueck, sonst None."""
    deps_by_id = {
        u["id"]: (u.get("deps") or [])
        for u in units
        if isinstance(u, dict) and "id" in u
    }
    WHITE, GRAY, BLACK = 0, 1, 2
    color = {uid: WHITE for uid in deps_by_id}
    stack = []

    def visit(uid):
        color[uid] = GRAY
        stack.append(uid)
        for dep in deps_by_id.get(uid, []):
            if dep not in deps_by_id:
                continue
            if color.get(dep) == GRAY:
                return stack[stack.index(dep) :] + [dep]
            if color.get(dep) == WHITE:
                found = visit(dep)
                if found:
                    return found
        stack.pop()
        color[uid] = BLACK
        return None

    for uid in deps_by_id:
        if color[uid] == WHITE:
            found = visit(uid)
            if found:
                return found
    return None


def run_preflight():
    errors = []
    files = [BACKLOG, BACKLOG_NEXT, BACKLOG_PARKED]
    loaded = {}  # Path -> units-Liste, nur fuer strukturell gueltige Dateien

    # Phase A: YAML-Parsing + Grundstruktur. Bei Verstoessen hier machen Cross-File-
    # Checks (Phase B) keinen Sinn mehr (Bezugsdaten fehlen) -- also gesammelt melden
    # und abbrechen, statt mit kaputten Daten weiterzurechnen.
    for path in files:
        data, err = load_yaml(path)
        if err:
            errors.append(err)
            continue
        units = units_of(data)
        if units is None:
            errors.append(f"{path.name}: kein Top-Level-Key 'units' mit einer Liste")
            continue
        for idx, u in enumerate(units):
            if not isinstance(u, dict) or "id" not in u or "status" not in u:
                errors.append(
                    f"{path.name}: Eintrag #{idx} hat kein 'id'- und/oder 'status'-Feld"
                )
        loaded[path] = units

    if len(loaded) < len(files):
        for e in errors:
            print(e, file=sys.stderr)
        sys.exit(1)

    # Phase B: Cross-File-Checks auf validierten Daten. Ab hier bewusst nicht mehr
    # beim ersten Fund abbrechen, sondern alles sammeln.
    id_owner = {}
    all_ids = set()
    for path, units in loaded.items():
        for u in units:
            if not isinstance(u, dict) or "id" not in u:
                continue
            uid = u["id"]
            if uid in id_owner:
                errors.append(f"Dublette ID '{uid}' in {id_owner[uid]} und {path.name}")
            else:
                id_owner[uid] = path.name
            all_ids.add(uid)

    # Regel 3: NUR BACKLOG.yml darf keine wartenden Units enthalten.
    for u in loaded[BACKLOG]:
        if not isinstance(u, dict):
            continue
        uid = u.get("id", "<ohne id>")
        ziel = "BACKLOG-PARKED.yml (verworfen/geparkt) oder BACKLOG-NEXT.yml (spaeterer Lauf)"
        if u.get("status") == "blocked":
            errors.append(
                f"BACKLOG.yml: Unit '{uid}' hat status: blocked -- gehoert nach {ziel}"
            )
        if "blocked_reason" in u:
            errors.append(
                f"BACKLOG.yml: Unit '{uid}' hat ein blocked_reason-Feld -- wartet auf eine Entscheidung, gehoert nach {ziel}"
            )
        done_when = u.get("done_when")
        if (
            isinstance(done_when, list)
            and done_when
            and LUKE_DECISION_RE.search(str(done_when[0]))
        ):
            errors.append(
                f"BACKLOG.yml: Unit '{uid}' hat done_when[0] als Entscheidung an Luke markiert -- gehoert nach {ziel}"
            )

    # Regel 4: deps-Referenzen ueber alle drei Dateien hinweg.
    for path, units in loaded.items():
        for u in units:
            if not isinstance(u, dict):
                continue
            uid = u.get("id", "<ohne id>")
            for dep in u.get("deps") or []:
                if dep not in all_ids:
                    errors.append(
                        f"{path.name}: Unit '{uid}' hat deps-Eintrag '{dep}', der in keiner der drei Dateien existiert"
                    )

    cycle = find_cycle(loaded[BACKLOG])
    if cycle:
        errors.append("BACKLOG.yml: Zyklus in deps: " + " -> ".join(cycle))

    if errors:
        for e in errors:
            print(e, file=sys.stderr)
        sys.exit(1)

    print("backlog-check: alle drei Dateien valide, keine Verstoesse.")
    sys.exit(0)


def run_state():
    data, err = load_yaml(BACKLOG)
    if err:
        print(err, file=sys.stderr)
        sys.exit(1)
    units = units_of(data)
    if units is None:
        print(
            f"{BACKLOG.name}: kein Top-Level-Key 'units' mit einer Liste",
            file=sys.stderr,
        )
        sys.exit(1)

    status_by_id = {
        u["id"]: u.get("status") for u in units if isinstance(u, dict) and "id" in u
    }
    open_count = sum(1 for s in status_by_id.values() if s in OPEN_STATUSES)

    next_id, model = "", "sonnet"
    for u in units:
        if not isinstance(u, dict) or u.get("status") != "todo":
            continue
        deps = u.get("deps") or []
        if not all(status_by_id.get(d) == "done" for d in deps):
            continue
        next_id = u.get("id", "")
        raw_model = u.get("model", "sonnet")
        if raw_model in VALID_MODELS:
            model = raw_model
        else:
            print(
                f"Warnung: Unit '{next_id}' hat unzulaessiges model '{raw_model}', falle auf sonnet zurueck",
                file=sys.stderr,
            )
        break

    print(f"OPEN={open_count}")
    print(f"NEXT={next_id}")
    print(f"MODEL={model}")
    sys.exit(0)


def run_model_of(unit_id):
    """Modell EINER bestimmten Unit. Der Treiber vergleicht damit nach jeder Iteration,
    ob die tatsaechlich gebaute Unit (aus der Journal-Ueberschrift) dasselbe Modell
    verlangt wie das, auf dem die Iteration lief -- die Messung zu B1.

    Sucht in allen drei Dateien: eine Unit kann im Lauf nach PARKED/NEXT gewandert sein.
    Unbekannte Unit -> MODEL= (leer), Exit 0; das ist kein Fehler, nur keine Aussage."""
    for path in (BACKLOG, BACKLOG_NEXT, BACKLOG_PARKED):
        data, err = load_yaml(path)
        if err:
            continue
        for u in units_of(data) or []:
            if isinstance(u, dict) and u.get("id") == unit_id:
                model = u.get("model", "sonnet")
                print("MODEL=%s" % (model if model in VALID_MODELS else "sonnet"))
                sys.exit(0)
    print("MODEL=")
    sys.exit(0)


def main():
    args = sys.argv[1:]
    if len(args) == 2 and args[0] == "--model-of":
        run_model_of(args[1])
        return
    if len(args) != 1 or args[0] not in ("--preflight", "--state"):
        print(
            "Verwendung: backlog-check.py --preflight | --state | --model-of <unit-id>",
            file=sys.stderr,
        )
        sys.exit(1)
    run_preflight() if args[0] == "--preflight" else run_state()


if __name__ == "__main__":
    main()
