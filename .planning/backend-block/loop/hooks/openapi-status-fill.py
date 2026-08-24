#!/usr/bin/env python3
"""Fuellt fehlende Status-Code-Response-Eintraege in backend/api/openapi.yaml.

Ersetzt das manuelle Abtippen von 4-10 YAML-Zeilen pro Nachtloop-Iteration
(27 von 29 Iterationen gingen dafuer drauf, ~3.3h nur fuer Spec-Zeilen). Quelle
ist der JSON-Dump, den TestOpenAPIStatusCodeDrift bei gesetzter
OPENAPI_DRIFT_DUMP-Variable schreibt (siehe
backend/internal/gateway/openapi_status_code_drift_test.go).

WICHTIG: reine textuelle Zeilen-Einfuegung, KEIN YAML-Round-Trip (49k Zeilen,
ein yaml.safe_load+dump wuerde alles umformatieren und den Diff unpruefbar
machen). Der Parser unten benutzt dieselben drei Regexe wie der Go-Test
(documentedStatusCodes), damit beide Seiten dieselbe Sicht auf die Datei haben.

Modi:
    --codes 400,503   (Default) Ziele kommen aus "missing" im Dump.
    --codes 409       Der Drift-Test kann 409 nicht melden (kommt aus der
                       Idempotency-Middleware, nie aus dem Handler-Body).
                       Ziele = alle mutierenden Operationen im Dump, abzueglich
                       Auth-Whitelist, WOPI-Routen und bereits dokumentiertem 409.
    --group PRAEFIX   Nur Operationen, deren Pfad mit PRAEFIX beginnt.
    --dry-run         Nur berichten, nichts schreiben.
    --file PFAD       Zieldatei, Default backend/api/openapi.yaml.

Aufruf: openapi-status-fill.py <dump.json> [--codes ...] [--group ...] [--dry-run] [--file ...]
"""
import json
import re
import sys
from pathlib import Path

# Pfade relativ zum Skript-Verzeichnis aufloesen, nicht zum Arbeitsverzeichnis
# (der Loop ruft aus wechselnden cwd auf).
REPO_ROOT = Path(__file__).resolve().parents[4]
DEFAULT_SPEC = REPO_ROOT / "backend" / "api" / "openapi.yaml"

# Dieselben drei Regexe wie backend/internal/gateway/openapi_status_code_drift_test.go
# (specPathKeyRE, specMethodKeyRE, specStatusKeyRE) -- absichtlich Zeichen fuer
# Zeichen identisch, sonst sehen Test und Skript unterschiedliche Dinge.
SPEC_PATH_KEY_RE = re.compile(r"^  (/\S+):\s*$")
SPEC_METHOD_KEY_RE = re.compile(r"^    (get|post|put|patch|delete|head|options):\s*$")
SPEC_STATUS_KEY_RE = re.compile(r"^        ['\"]?(\d{3})['\"]?:")

# Komponentennamen fuer die einzufuegenden $ref-Antworten (bereits definiert
# in api/openapi.yaml components/responses).
REF_FOR_CODE = {
    400: "BadRequest",
    503: "ServiceUnavailable",
    409: "IdempotencyInFlight",
}

# lean: 409-Ausschlussliste hart im Code, weil sie drei tatsaechliche
# Codepruefungen zusammenfasst (nicht geraten): idempotencyWhitelist in
# backend/internal/middleware/idempotency.go:36 (strings.Contains-Match, exakt
# diese drei Substrings), WOPI-Routen liegen unter /wopi/... und werden ueber
# RegisterRoutes(r, nil) OHNE die Idempotency-Middleware registriert
# (cmd/gateway/main.go:349) -- erscheinen im Dump aktuell ohnehin nicht, weil
# der Drift-Test nur /api/v1/*-Routen prueft, aber der Filter bleibt explizit
# statt sich auf diesen Zufall zu verlassen. Upgrade-Trigger: wird die
# Idempotency-Middleware auf weitere Pfad-Praefixe oder Nicht-/api/v1-Routen
# ausgeweitet, muss diese Liste mitgezogen werden.
IDEMPOTENCY_WHITELIST_SUBSTRINGS = ("/auth/login", "/auth/refresh", "/auth/2fa")
WOPI_PATH_PREFIX = "/wopi/"


def parse_spec(lines):
    """Baut einen Index ueber alle Operationen des paths:-Blocks: "METHOD /pfad"
    -> {responses_line, entries: [(code, zeilenindex), ...], block_end}.
    Exakt derselbe Automat wie documentedStatusCodes im Go-Test, nur dass hier
    zusaetzlich die Zeilenindizes mitgefuehrt werden (der Test braucht nur die
    Codes, wir brauchen die Positionen fuer die Einfuegung)."""
    ops = {}
    in_paths = False
    in_responses = False
    current_path = ""
    current_op_key = None
    n = len(lines)

    def end_responses(line_idx):
        nonlocal in_responses
        if in_responses and current_op_key is not None:
            op = ops[current_op_key]
            if op["block_end"] is None:
                op["block_end"] = line_idx
        in_responses = False

    end_of_paths = n  # Zeile, an der der paths:-Block endet (0-Indent-Key wie
    # "components:", oder Dateiende). Fuer die letzte Operation gibt es keinen
    # naechsten Path-/Method-Key, der ihren responses:-Block beendet -- ohne
    # dieses Tracking wuerde end_responses() unten mit n (Dateiende) statt der
    # echten Grenze aufgerufen und eine Einfuegung liefe mitten in
    # components: hinein (so beobachtet: eine 503-Zeile landete im
    # ServiceUnavailable-Response-Objekt selbst).
    for i in range(n):
        line = lines[i].rstrip("\r\n")
        if not in_paths:
            if line.startswith("paths:"):
                in_paths = True
            continue
        if line != "" and line[0] not in (" ", "\t"):
            end_of_paths = i
            break

        m = SPEC_PATH_KEY_RE.match(line)
        if m:
            end_responses(i)
            current_path = m.group(1)
            current_op_key = None
            continue

        m = SPEC_METHOD_KEY_RE.match(line)
        if m:
            end_responses(i)
            current_op_key = m.group(1).upper() + " " + current_path
            ops[current_op_key] = {
                "responses_line": None,
                "entries": [],
                "block_end": None,
            }
            continue

        if current_op_key is None:
            continue

        if line == "      responses:":
            in_responses = True
            ops[current_op_key]["responses_line"] = i
            continue

        if (
            in_responses
            and line.startswith("      ")
            and len(line) > 6
            and line[6] != " "
        ):
            end_responses(i)

        if not in_responses:
            continue

        m = SPEC_STATUS_KEY_RE.match(line)
        if m:
            ops[current_op_key]["entries"].append((int(m.group(1)), i))

    end_responses(end_of_paths)
    return ops


def path_group(path):
    """Registrar-Gruppe eines Pfads fuer die Zusammenfassung, z.B.
    "/api/v1/tags/{id}" -> "/api/v1/tags"."""
    parts = path.split("/")
    return "/".join(parts[:4]) or path


def targets_for_code(dump_ops, code, group):
    """Liefert die (METHOD, path)-Ziele fuer einen Status-Code, plus bei Code
    409 die Anzahl der aus Whitelist/WOPI ausgeschlossenen mutierenden
    Operationen (fuer den Bericht)."""
    excluded = {"whitelist": 0, "wopi": 0, "already_documented": 0}
    result = []
    for op in dump_ops:
        path = op["path"]
        if group and not path.startswith(group):
            continue
        if code == 409:
            if not op["mutating"]:
                continue
            if 409 in op["documented"]:
                excluded["already_documented"] += 1
                continue
            if any(s in path for s in IDEMPOTENCY_WHITELIST_SUBSTRINGS):
                excluded["whitelist"] += 1
                continue
            if path.startswith(WOPI_PATH_PREFIX):
                excluded["wopi"] += 1
                continue
        else:
            if code not in op["missing"]:
                continue
        result.append((op["method"], path))
    return result, excluded


def last_content_line_before(block_end, lines):
    """Letzte echte Response-Body-Zeile vor block_end. Ueberspringt Leerzeilen
    UND alles mit weniger als 8 Leerzeichen Einrueckung -- das sind
    zwangslaeufig Kommentare zwischen zwei Operationen (497 Stueck im File,
    z.B. "  # Contacts" als Abschnittsueberschrift vor der naechsten
    Pfadgruppe), nie echter Response-Inhalt, der liegt immer bei >= 8. Ohne
    diesen Filter landet eine Einfuegung hinter so einem Kommentar statt davor."""
    idx = block_end - 1
    while idx >= 0:
        line = lines[idx].rstrip("\r\n")
        stripped = line.strip()
        if stripped == "":
            idx -= 1
            continue
        indent = len(line) - len(line.lstrip(" "))
        if indent < 8:
            idx -= 1
            continue
        break
    return idx


def plan_insertions(spec_ops, wanted, lines):
    """wanted: Liste aus (method, path, code). Gibt (insertions, skipped)
    zurueck. insertions: Liste aus (zeilenindex, [neue_zeilen]), noch nicht
    nach Position gruppiert. skipped: Liste aus ((method, path, code), grund)."""
    # Neue Codes pro Operation buendeln, damit zwei gleichzeitig eingefuegte
    # Codes derselben Operation nicht am selben Punkt in falscher Reihenfolge
    # landen.
    by_op = {}
    skipped = []
    for method, path, code in wanted:
        op_key = f"{method} {path}"
        op = spec_ops.get(op_key)
        if op is None:
            skipped.append(
                ((method, path, code), "Operation nicht in der Spec gefunden")
            )
            continue
        if op["responses_line"] is None or op["block_end"] is None:
            skipped.append(((method, path, code), "responses:-Block nicht gefunden"))
            continue
        if any(c == code for c, _ in op["entries"]):
            skipped.append(((method, path, code), "Code bereits dokumentiert"))
            continue
        by_op.setdefault(op_key, []).append(code)

    insertions = []
    for op_key, codes in by_op.items():
        op = spec_ops[op_key]
        groups = {}  # Ziel-Zeilenindex -> Liste neuer Codes
        for code in codes:
            anchor = next((idx for c, idx in op["entries"] if c > code), None)
            if anchor is None:
                anchor = last_content_line_before(op["block_end"], lines) + 1
            groups.setdefault(anchor, []).append(code)
        for anchor, group_codes in groups.items():
            new_lines = []
            for code in sorted(group_codes):
                new_lines.extend(entry_lines(code, newline_of(lines)))
            insertions.append((anchor, new_lines))

    return insertions, skipped


def newline_of(lines):
    """Zeilenende-Konvention der Datei (durchgaengig CRLF im Repo, aber nicht
    fest verdrahtet)."""
    for line in lines:
        if line.endswith("\r\n"):
            return "\r\n"
        if line.endswith("\n"):
            return "\n"
    return "\n"


def entry_lines(code, newline):
    ref = REF_FOR_CODE[code]
    return [
        f'        "{code}":{newline}',
        f'          $ref: "#/components/responses/{ref}"{newline}',
    ]


def apply_insertions(lines, insertions):
    """Wendet die Einfuegungen von unten nach oben an (absteigende
    Zeilennummer), damit vorher berechnete Positionen gueltig bleiben."""
    for anchor, new_lines in sorted(insertions, key=lambda x: x[0], reverse=True):
        lines[anchor:anchor] = new_lines


def parse_args(argv):
    if not argv or argv[0].startswith("--"):
        print(
            "Verwendung: openapi-status-fill.py <dump.json> [--codes 400,503|409] "
            "[--group PRAEFIX] [--dry-run] [--file PFAD]",
            file=sys.stderr,
        )
        sys.exit(1)
    opts = {
        "dump": Path(argv[0]),
        "codes": [400, 503],
        "group": None,
        "dry_run": False,
        "file": DEFAULT_SPEC,
    }
    i = 1
    while i < len(argv):
        arg = argv[i]
        if arg == "--codes":
            i += 1
            opts["codes"] = [int(c) for c in argv[i].split(",")]
        elif arg == "--group":
            i += 1
            opts["group"] = argv[i]
        elif arg == "--dry-run":
            opts["dry_run"] = True
        elif arg == "--file":
            i += 1
            opts["file"] = Path(argv[i])
        else:
            print(f"Unbekanntes Argument: {arg}", file=sys.stderr)
            sys.exit(1)
        i += 1
    for code in opts["codes"]:
        if code not in REF_FOR_CODE:
            print(
                f"Kein bekannter Response-Component fuer Code {code} (bekannt: {sorted(REF_FOR_CODE)})",
                file=sys.stderr,
            )
            sys.exit(1)
    return opts


def main():
    opts = parse_args(sys.argv[1:])

    with open(opts["dump"], encoding="utf-8") as f:
        dump_ops = json.load(f)["operations"]

    spec_path = opts["file"]
    with open(spec_path, encoding="utf-8", newline="") as f:
        lines = f.readlines()

    spec_ops = parse_spec(lines)

    wanted = []
    exclusion_report = []
    for code in opts["codes"]:
        targets, excluded = targets_for_code(dump_ops, code, opts["group"])
        for method, path in targets:
            wanted.append((method, path, code))
        if code == 409 and any(excluded.values()):
            exclusion_report.append((code, excluded))

    insertions, skipped = plan_insertions(spec_ops, wanted, lines)

    inserted_count_by_code = {}
    inserted_count_by_group = {}
    for _, new_lines in insertions:
        for line in new_lines:
            m = SPEC_STATUS_KEY_RE.match(line.rstrip("\r\n"))
            if m:
                inserted_count_by_code[int(m.group(1))] = (
                    inserted_count_by_code.get(int(m.group(1)), 0) + 1
                )

    # Gruppenzaehlung ueber die tatsaechlichen wanted-Ziele, nicht ueber die
    # gebuendelten Insertions (eine Gruppe soll die Anzahl neuer Eintraege
    # zeigen, nicht die Anzahl Einfuegepunkte).
    applied_targets = {(m, p, c) for m, p, c in wanted} - {t for t, _ in skipped}
    for method, path, code in applied_targets:
        grp = path_group(path)
        inserted_count_by_group.setdefault(grp, {}).setdefault(code, 0)
        inserted_count_by_group[grp][code] += 1

    if not opts["dry_run"]:
        apply_insertions(lines, insertions)
        with open(spec_path, "w", encoding="utf-8", newline="") as f:
            f.writelines(lines)

    # --------------------------------------------------------------- Bericht
    mode = (
        "DRY-RUN (nichts geschrieben)"
        if opts["dry_run"]
        else f"angewendet auf {spec_path}"
    )
    print(f"openapi-status-fill: {mode}")
    print(
        f"Codes: {opts['codes']}"
        + (f", Gruppe: {opts['group']}" if opts["group"] else "")
    )
    print()
    print("Eingefuegt:")
    if inserted_count_by_code:
        for code in sorted(inserted_count_by_code):
            print(f"  {code}: {inserted_count_by_code[code]} Eintraege")
    else:
        print("  (keine)")
    print()
    print("Je Pfadgruppe:")
    for grp in sorted(inserted_count_by_group):
        per_code = ", ".join(
            f"{c}={n}" for c, n in sorted(inserted_count_by_group[grp].items())
        )
        print(f"  {grp}: {per_code}")
    if not inserted_count_by_group:
        print("  (keine)")

    for code, excluded in exclusion_report:
        print()
        print("409-Ausschluesse (nicht als Ziel gewertet, kein Fehler):")
        print(f"  Auth-Whitelist (login/refresh/2fa): {excluded['whitelist']}")
        print(f"  WOPI-Routen: {excluded['wopi']}")
        print(f"  bereits dokumentiertes 409: {excluded['already_documented']}")

    print()
    print(f"Uebersprungen (nicht anwendbar): {len(skipped)}")
    for (method, path, code), reason in skipped:
        print(f"  {method} {path} ({code}): {reason}")

    sys.exit(0)


if __name__ == "__main__":
    main()
