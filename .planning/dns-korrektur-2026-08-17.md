# DNS-Korrektur zentria.tech — Anleitung

> Stand 2026-08-17. Auszuführen in der **Hetzner DNS-Konsole** (https://dns.hetzner.com), Zone
> `zentria.tech`. Von Hand, weil Hetzners Konsole Browser-Automatisierung blockt (Bot-Schutz bleibt
> bei „Verifying" hängen).
>
> Vier Änderungen. **Schritt 1 ist der wichtige und der einzige mit Risiko** — die drei Löschungen
> danach sind folgenlos.

---

## Warum überhaupt

Beim Aufräumen der toten Resend-Records ist aufgefallen, dass Produktion über **Brevo** versendet:

```
SYSTEM_SMTP_HOST=smtp-relay.brevo.com     (aus /opt/kmuhub/.env.production)
SYSTEM_SMTP_FROM=noreply@zentria.tech
```

Der SPF-Record der Domain erlaubt aber **nur Strato**:

```
zentria.tech.  TXT  "v=spf1 redirect=_spf.strato.com"
_spf.strato.com → ip4:81.169.146.128/25 … -all      ← Hard Fail, Brevo nicht enthalten
_dmarc.zentria.tech. TXT "v=DMARC1;p=reject;"
```

Jede Mail, die das Produkt verschickt — Passwort-Reset, Einladungen — fällt damit bei SPF **hart
durch**. Durchkommen kann sie nur noch über DKIM-Alignment (`brevo1/brevo2._domainkey` sind
gesetzt). Selbst wenn das greift, ist ein SPF-Hardfail bei vielen Empfängern ein Spam-Signal.

Das ist ein plausibler Grund, warum der Reset-Flow nie funktionierend gesehen wurde.

---

## Schritt 1 — SPF um Brevo erweitern (wichtig, mit Risiko)

**Record finden:** Typ `TXT`, Name `@` (oder leer / `zentria.tech`), Wert beginnt mit `v=spf1`.

**Alt:**
```
v=spf1 redirect=_spf.strato.com
```

**Neu:**
```
v=spf1 include:_spf.strato.com include:spf.brevo.com -all
```

**Warum nicht einfach `include:spf.brevo.com` anhängen:** `redirect=` ersetzt die gesamte Auswertung
und darf nicht neben `include`/`all` stehen. Deshalb wird `redirect=` zu `include:` und das `-all`
kommt explizit ans Ende — genau das `-all`, das vorher aus Stratos Record kam.

**Risiko:** Falsch gesetzt brechen die sieben Strato-Postfächer (ausgehend). Deshalb den Wert
zeichengenau übernehmen, insbesondere den Unterstrich in `_spf.strato.com` und das Minus in `-all`.

**Lookup-Budget:** 2 von 10 erlaubten DNS-Lookups. Beide Includes lösen direkt auf IP-Listen auf,
verschachteln also nicht weiter.

**Verifikation** (Windows/PowerShell oder Git-Bash, nach TTL-Ablauf):
```
nslookup -type=TXT zentria.tech 8.8.8.8
```
Erwartet: `"v=spf1 include:_spf.strato.com include:spf.brevo.com -all"`

---

## Schritt 2–4 — tote Resend-Records löschen (risikofrei)

Resend ist restlos aus dem Website-Code entfernt; die Website versendet gar keine Mails mehr. Diese
drei Einträge zeigen ins Leere — einer davon auf **Amazon SES**, also einen US-Dienst in der Zone
einer Domain, die mit „keine Auftragsverarbeiter außerhalb der EU" wirbt. Wer die Zone abfragt,
findet ihn.

| # | Typ | Name | Aktueller Wert | Aktion |
|---|---|---|---|---|
| 2 | `MX` | `send` | `feedback-smtp.eu-west-1.amazonses.com` (Prio 10) | löschen |
| 3 | `TXT` | `send` | SPF-Eintrag für den Resend-Versand | löschen |
| 4 | `TXT` | `resend._domainkey` | Resend-DKIM-Schlüssel | löschen |

**Verifikation:**
```
nslookup -type=MX send.zentria.tech 8.8.8.8      → NXDOMAIN
nslookup -type=TXT resend._domainkey.zentria.tech 8.8.8.8   → NXDOMAIN
```

---

## NICHT anfassen

| Record | Grund |
|---|---|
| `brevo1._domainkey` CNAME → `b1.zentria-tech.dkim.brevo.com` | **aktiv** — signiert die Produkt-Mails |
| `brevo2._domainkey` CNAME → `b2.zentria-tech.dkim.brevo.com` | **aktiv** — dito |
| `@` MX → `smtpin.rzone.de` | Posteingang der sieben Strato-Postfächer |
| `_dmarc` TXT `p=reject` | soll scharf bleiben |
| `@`, `www`, `app`, `s3` A-Records → `178.104.38.195` | Website und Produkt |

---

## Danach: der eigentliche Test

Der Mailempfang ist seit gestern belegt, der **Versand** noch nicht. Sobald Schritt 1 durch ist,
lässt sich der Reset-Flow zum ersten Mal Ende zu Ende prüfen — er steht seit Wochen als offener
Posten:

1. Auf `app.zentria.tech` „Passwort vergessen" mit einer echten Adresse auslösen.
2. Prüfen, ob die Mail ankommt — und **im Quelltext der Mail** nachsehen:
   `Authentication-Results:` sollte `spf=pass` und `dkim=pass` zeigen. Vorher wäre dort `spf=fail`
   gestanden.
3. Link anklicken, Passwort setzen, mit dem neuen Passwort anmelden.

Landet die Mail im Spam, ist der SPF-Fix genau der Grund, warum sie es vorher tat.
