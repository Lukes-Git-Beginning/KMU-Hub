# Review-Fäden — buchhaltung (finanzen, Backend-Themen Strom L)

> Eine Datei pro Modul. Der Bau-Strom trägt nach JEDER fertigen Phase einen Eintrag ein (Struktur siehe `_TEMPLATE.md`). Vorlage für Dariens Feinschliff-Review (Pfad nachklicken, worauf achten, Screenshots, offene Punkte).

**Modul:** `buchhaltung` · **Strom:** L (Backend) · **Reviewer (zugeteilt):** offen

---

## Backend-Welle 2 + Smoke-Verifikation (2026-06-10)  ·  main `3761bcd6`…`f422cfcb`  ·  Status: ⬜ ungereviewt

**Was passiert ist (Kurzfassung für den Review):**
- GoBD-Belegarchiv (Migr. 139), E-Rechnung-Eingang (Migr. 140), Kontakte-360°-Filter (Migr. 141) live auf Production.
- Proto-Vollregen reaktivierte 10 nie-funktionale RPCs; Production-Smoke bewies LockInvoice (RE-2026-0001, absichtlich dauerhaft festgeschrieben) + GoBD-Export end-to-end.
- Dabei 2 echte Production-Bugs gefunden + gefixt: PDF-RPC-nil-Panic ohne company_settings (`b0653fef`/`8708f30a`, jetzt HTTP 410) und fehlende Tenant-Metadata am biz→crm-gRPC-Client (`f422cfcb`, QuoteFromDeal war 401).

**Offene Fragen für Darien (Domäne/Produkt):**
1. **Eingangsrechnungs-Workflow nach `reviewed`:** `finance_incoming_invoices` endet aktuell bei received→reviewed→booked/rejected. Was passiert fachlich nach `reviewed` — Buchung in Cosmi (Vorkette) oder direkte Übergabe an DATEV-Export? Soll `booked` einen DATEV-Stapel-Eintrag erzeugen?
2. **GoBD-Belegarchiv nur für gelockte Invoices ok?** `ArchiveInvoiceDocument` verlangt aktuell eine festgeschriebene Rechnung. Sollen auch Eingangsbelege/sonstige Dokumente (ohne Lock-Konzept) archivierbar sein? (Route `POST /finance/gobd-archive` existiert generisch — Abgrenzung klären.)
3. **ZUGFeRD-200-Vollbeweis:** braucht konfigurierte `company_settings` am Prod-Tenant — wer pflegt die echten Zentria-/Pilot-Firmendaten ein (Onboarding-Schritt)?

**Reviewer-Notizen (beim Feinschliff auszufüllen):**
- _<hier trägt der Reviewer Feedback ein → wird zu TaskCreate-Items>_

---

<!-- Phasen-Einträge hier anhängen — Struktur aus _TEMPLATE.md kopieren.
     Status-Legende: ⬜ ungereviewt · 🟡 Feedback offen · ✅ grün -->
