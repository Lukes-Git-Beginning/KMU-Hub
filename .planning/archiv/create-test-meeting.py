#!/usr/bin/env python3
"""
Legt das Videocall-Test-Meeting an und startet es — komplett ueber DEINEN Account.
Nur dein Login wird abgefragt (kein Passwort gespeichert/geteilt). Die beiden
Attendees (Darien + Nico) sind als UUID fest hinterlegt, daher KEINE Admin-Rolle
und keine User-Suche noetig — jede Rolle mit meetings:write reicht (alle haben das).

Aufruf (Git Bash oder PowerShell), aus dem Repo-Root:
    python .planning/create-test-meeting.py
Optionaler Titel:
    python .planning/create-test-meeting.py --title "Kunden-Demo Call"
"""
import sys, json, getpass, uuid, urllib.request, urllib.error
from datetime import datetime, timedelta, timezone

BASE = "https://app.zentria.tech"

# Attendees (aus der Prod-DB, gleicher Tenant) — Luke wird als Organisator
# automatisch ergaenzt.
ATTENDEE_IDS = [
    "4ba037b7-5ec1-454d-b6ca-0a6f3dc9c66e",  # darien.stelter@googlemail.com
    "2306bfa8-9e0e-4537-975e-f01c6049d45c",  # laramariehartig@gmail.com (Nico)
]


def call(method, path, token=None, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(BASE + path, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    if token:
        req.add_header("Authorization", "Bearer " + token)
    # Gateway runs idempotency in hard mode: POST/PUT/PATCH/DELETE need a key.
    if method.upper() != "GET":
        req.add_header("Idempotency-Key", str(uuid.uuid4()))
    try:
        with urllib.request.urlopen(req, timeout=20) as r:
            raw = r.read().decode()
            return r.status, (json.loads(raw) if raw.strip() else {})
    except urllib.error.HTTPError as e:
        raw = e.read().decode()
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"raw": raw}


def pick(d, *keys):
    for k in keys:
        if isinstance(d, dict) and d.get(k) not in (None, ""):
            return d[k]
    return None


def main():
    title = "Videocall-Test"
    if "--title" in sys.argv:
        title = sys.argv[sys.argv.index("--title") + 1]

    email = input("Deine Login-E-Mail: ").strip()
    password = getpass.getpass("Dein Passwort: ")

    # 1) Login
    st, resp = call(
        "POST", "/api/v1/auth/login", body={"email": email, "password": password}
    )
    if st != 200:
        print(f"Login fehlgeschlagen ({st}): {resp}")
        sys.exit(1)
    if pick(resp, "pending_token", "pendingToken") and not pick(
        resp, "access_token", "accessToken", "token"
    ):
        print("2FA aktiv — Skript unterstuetzt kein TOTP. Sag Bescheid.")
        sys.exit(1)
    token = pick(resp, "access_token", "accessToken", "token")
    if not token:
        print(f"Kein Token gefunden. Response-Keys: {list(resp.keys())}")
        sys.exit(1)
    print("Login OK.")

    # 2) Meeting anlegen
    now = datetime.now(timezone.utc)
    body = {
        "title": title,
        "scheduled_start": now.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "scheduled_end": (now + timedelta(hours=2)).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "attendee_user_ids": ATTENDEE_IDS,
    }
    st, resp = call("POST", "/api/v1/meetings", token=token, body=body)
    if st not in (200, 201):
        print(f"Meeting-Erstellen fehlgeschlagen ({st}): {resp}")
        sys.exit(1)
    meeting = resp.get("meeting", resp) if isinstance(resp, dict) else resp
    mid = pick(meeting, "id", "meeting_id", "meetingId")
    if not mid:
        print(f"Keine Meeting-ID im Response: {resp}")
        sys.exit(1)
    print(f"Meeting erstellt: {mid}")

    # 3) Starten -> in_progress (damit 'Beitreten' bei allen erscheint)
    st, resp = call("POST", f"/api/v1/meetings/{mid}/start", token=token)
    if st != 200:
        print(f"Start fehlgeschlagen ({st}): {resp}")
        sys.exit(1)
    print("\n=== FERTIG ===")
    print(f"Meeting '{title}' ist LIVE (id={mid}).")
    print("Alle drei: App -> Meetings -> 'Live jetzt' -> 'Beitreten'.")


if __name__ == "__main__":
    main()
