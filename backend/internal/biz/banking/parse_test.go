package banking

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

const camtSample = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <GrpHdr><MsgId>MSG-1</MsgId><CreDtTm>2026-02-01T06:00:00</CreDtTm></GrpHdr>
    <Stmt>
      <Id>STMT-2026-01</Id>
      <CreDtTm>2026-02-01T06:00:00</CreDtTm>
      <Acct><Id><IBAN>DE02120300000000202051</IBAN></Id><Ccy>EUR</Ccy></Acct>
      <Bal>
        <Tp><CdOrPrtry><Cd>OPBD</Cd></CdOrPrtry></Tp>
        <Amt Ccy="EUR">1000.00</Amt><CdtDbtInd>CRDT</CdtDbtInd>
        <Dt><Dt>2026-01-01</Dt></Dt>
      </Bal>
      <Bal>
        <Tp><CdOrPrtry><Cd>CLBD</Cd></CdOrPrtry></Tp>
        <Amt Ccy="EUR">1069.05</Amt><CdtDbtInd>CRDT</CdtDbtInd>
        <Dt><Dt>2026-01-31</Dt></Dt>
      </Bal>
      <Ntry>
        <NtryRef>ENTRY-1</NtryRef>
        <Amt Ccy="EUR">119.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <Sts>BOOK</Sts>
        <BookgDt><Dt>2026-01-15</Dt></BookgDt>
        <ValDt><Dt>2026-01-16</Dt></ValDt>
        <NtryDtls><TxDtls>
          <Refs><EndToEndId>E2E-42</EndToEndId></Refs>
          <RltdPties>
            <Dbtr><Nm>Muster GmbH</Nm></Dbtr>
            <DbtrAcct><Id><IBAN>DE89370400440532013000</IBAN></Id></DbtrAcct>
          </RltdPties>
          <RmtInf><Ustrd>Zahlung Rechnung RE-2026-0001</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
      <Ntry>
        <NtryRef>ENTRY-2</NtryRef>
        <Amt Ccy="EUR">49.95</Amt>
        <CdtDbtInd>DBIT</CdtDbtInd>
        <Sts>BOOK</Sts>
        <ValDt><Dt>2026-01-20</Dt></ValDt>
        <NtryDtls><TxDtls>
          <RltdPties>
            <Cdtr><Nm>Stadtwerke</Nm></Cdtr>
            <CdtrAcct><Id><IBAN>DE11520513735120710131</IBAN></Id></CdtrAcct>
          </RltdPties>
          <RmtInf><Ustrd>Abschlag Strom</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
      <Ntry>
        <NtryRef>ENTRY-3</NtryRef>
        <Amt Ccy="EUR">500.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <Sts>PDNG</Sts>
        <ValDt><Dt>2026-01-31</Dt></ValDt>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

func TestParseCAMT053(t *testing.T) {
	stmt, err := Parse([]byte(camtSample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if stmt.Format != BankFormatCAMT053 {
		t.Errorf("format = %q, want %q", stmt.Format, BankFormatCAMT053)
	}
	if stmt.AccountIBAN != "DE02120300000000202051" {
		t.Errorf("account = %q", stmt.AccountIBAN)
	}
	if stmt.StatementRef != "STMT-2026-01" {
		t.Errorf("statement ref = %q", stmt.StatementRef)
	}
	if stmt.Currency != "EUR" {
		t.Errorf("currency = %q", stmt.Currency)
	}
	if stmt.OpeningBalance == nil || !stmt.OpeningBalance.Equal(decimal.RequireFromString("1000.00")) {
		t.Errorf("opening balance = %v", stmt.OpeningBalance)
	}
	if stmt.ClosingBalance == nil || !stmt.ClosingBalance.Equal(decimal.RequireFromString("1069.05")) {
		t.Errorf("closing balance = %v", stmt.ClosingBalance)
	}

	// The pending entry is skipped: only booked entries may be reconciled.
	if len(stmt.Entries) != 2 {
		t.Fatalf("entries = %d, want 2 (pending entry must be skipped)", len(stmt.Entries))
	}

	credit := stmt.Entries[0]
	if !credit.Amount.Equal(decimal.RequireFromString("119.00")) {
		t.Errorf("credit amount = %s, want 119.00", credit.Amount)
	}
	if credit.ValueDate != time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC) {
		t.Errorf("value date = %v", credit.ValueDate)
	}
	if credit.BookingDate == nil || *credit.BookingDate != time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC) {
		t.Errorf("booking date = %v", credit.BookingDate)
	}
	if credit.EndToEndID != "E2E-42" {
		t.Errorf("end-to-end id = %q", credit.EndToEndID)
	}
	// On money in, the counterparty is the debtor.
	if credit.CounterpartyName != "Muster GmbH" {
		t.Errorf("counterparty = %q, want the debtor on a credit", credit.CounterpartyName)
	}
	if credit.RemittanceInfo != "Zahlung Rechnung RE-2026-0001" {
		t.Errorf("remittance = %q", credit.RemittanceInfo)
	}

	debit := stmt.Entries[1]
	if !debit.Amount.Equal(decimal.RequireFromString("-49.95")) {
		t.Errorf("debit amount = %s, want -49.95 (DBIT must be negative)", debit.Amount)
	}
	// On money out, the counterparty is the creditor.
	if debit.CounterpartyName != "Stadtwerke" {
		t.Errorf("debit counterparty = %q, want the creditor on a debit", debit.CounterpartyName)
	}
}

func TestParseCAMTRejectsMissingIndicator(t *testing.T) {
	// An entry without CdtDbtInd has no readable direction. Defaulting to credit
	// would book an outgoing transfer as an incoming payment.
	broken := strings.Replace(camtSample, "<CdtDbtInd>CRDT</CdtDbtInd>\n        <Sts>BOOK</Sts>", "<Sts>BOOK</Sts>", 1)
	if _, err := Parse([]byte(broken)); !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed", err)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    error
	}{
		{"empty", "", ErrEmptyFile},
		{"plain text", "just some notes about the bank", ErrUnknownFormat},
		{"truncated xml", "<Document><BkToCstmrStmt><Stmt>", ErrMalformed},
		{"xml without statement", `<Document xmlns="x"><BkToCstmrStmt></BkToCstmrStmt></Document>`, ErrMalformed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.content))
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestParseRejectsOversizedFile(t *testing.T) {
	if _, err := Parse(make([]byte, MaxStatementBytes+1)); !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("err = %v, want ErrFileTooLarge", err)
	}
}

const mt940Sample = ":20:STARTUMS\r\n" +
	":25:DE02120300000000202051/EUR\r\n" +
	":28C:00001/001\r\n" +
	":60F:C260101EUR1000,00\r\n" +
	":61:2601160115CR119,00NTRFE2E-42//BANKREF-1\r\n" +
	":86:166?00GUTSCHRIFT?20Zahlung Rechnung RE-?212026-0001?31DE89370400440532013000?32Muster GmbH\r\n" +
	":61:2601200120DR49,95NDDTNONREF//BANKREF-2\r\n" +
	":86:005?00LASTSCHRIFT?20Abschlag Strom?32Stadtwerke\r\n" +
	":62F:C260131EUR1069,05\r\n"

func TestParseMT940(t *testing.T) {
	stmt, err := Parse([]byte(mt940Sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if stmt.Format != BankFormatMT940 {
		t.Errorf("format = %q", stmt.Format)
	}
	if stmt.AccountIBAN != "DE02120300000000202051" {
		t.Errorf("account = %q (the /EUR suffix must be stripped)", stmt.AccountIBAN)
	}
	if stmt.Currency != "EUR" {
		t.Errorf("currency = %q", stmt.Currency)
	}
	if stmt.OpeningBalance == nil || !stmt.OpeningBalance.Equal(decimal.RequireFromString("1000")) {
		t.Errorf("opening balance = %v", stmt.OpeningBalance)
	}
	if stmt.ClosingBalance == nil || !stmt.ClosingBalance.Equal(decimal.RequireFromString("1069.05")) {
		t.Errorf("closing balance = %v", stmt.ClosingBalance)
	}
	if len(stmt.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(stmt.Entries))
	}

	credit := stmt.Entries[0]
	if !credit.Amount.Equal(decimal.RequireFromString("119.00")) {
		t.Errorf("credit amount = %s, want 119.00 (comma is the decimal separator)", credit.Amount)
	}
	if credit.ValueDate != time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC) {
		t.Errorf("value date = %v", credit.ValueDate)
	}
	if credit.BookingDate == nil || *credit.BookingDate != time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC) {
		t.Errorf("booking date = %v", credit.BookingDate)
	}
	if credit.Currency != "EUR" {
		t.Errorf("entry currency = %q, want the statement currency", credit.Currency)
	}
	// ?20 and ?21 are one invoice number split across two chunks: joining them
	// with a separator would break the reference.
	if credit.RemittanceInfo != "Zahlung Rechnung RE-2026-0001" {
		t.Errorf("remittance = %q, want the ?20/?21 chunks joined without a separator", credit.RemittanceInfo)
	}
	if credit.CounterpartyName != "Muster GmbH" {
		t.Errorf("counterparty = %q", credit.CounterpartyName)
	}
	if credit.CounterpartyIBAN != "DE89370400440532013000" {
		t.Errorf("counterparty iban = %q", credit.CounterpartyIBAN)
	}
	if credit.EntryRef != "BANKREF-1" {
		t.Errorf("entry ref = %q, want the bank reference", credit.EntryRef)
	}
	if credit.EndToEndID != "E2E-42" {
		t.Errorf("end-to-end id = %q", credit.EndToEndID)
	}

	debit := stmt.Entries[1]
	if !debit.Amount.Equal(decimal.RequireFromString("-49.95")) {
		t.Errorf("debit amount = %s, want -49.95 (D must be negative)", debit.Amount)
	}
	if debit.EndToEndID != "" {
		t.Errorf("end-to-end id = %q, want NONREF to resolve to empty", debit.EndToEndID)
	}
}

func TestParseMT940ReversalFlipsSign(t *testing.T) {
	// RC reverses a credit, so money leaves the account; RD reverses a debit.
	content := ":20:REV\r\n:60F:C260101EUR0,00\r\n" +
		":61:260116RC119,00NTRFNONREF\r\n" +
		":61:260117RD49,95NTRFNONREF\r\n"
	stmt, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmt.Entries) != 2 {
		t.Fatalf("entries = %d", len(stmt.Entries))
	}
	if !stmt.Entries[0].Amount.Equal(decimal.RequireFromString("-119.00")) {
		t.Errorf("RC amount = %s, want -119.00", stmt.Entries[0].Amount)
	}
	if !stmt.Entries[1].Amount.Equal(decimal.RequireFromString("49.95")) {
		t.Errorf("RD amount = %s, want 49.95", stmt.Entries[1].Amount)
	}
}

func TestParseMT940BookingDateAcrossYearEnd(t *testing.T) {
	// Booked 30 December, valued 2 January: the booking belongs to the previous
	// year, not to the value date's.
	content := ":20:YEAREND\r\n:60F:C260101EUR0,00\r\n:61:2601021230CR10,00NTRFNONREF\r\n"
	stmt, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := stmt.Entries[0].BookingDate
	want := time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC)
	if got == nil || *got != want {
		t.Fatalf("booking date = %v, want %v", got, want)
	}
}

func TestParseMT940RejectsUnreadableLine(t *testing.T) {
	content := ":20:BROKEN\r\n:60F:C260101EUR0,00\r\n:61:this is not a booking line\r\n"
	if _, err := Parse([]byte(content)); !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed", err)
	}
}

func TestParseMT940UnstructuredInformation(t *testing.T) {
	// A bank that does not use "?NN" subfields: the whole field is remittance.
	content := ":20:PLAIN\r\n:60F:C260101EUR0,00\r\n" +
		":61:260116CR119,00NTRFNONREF\r\n" +
		":86:Rechnung RE-2026-0001 Muster GmbH\r\n"
	stmt, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := stmt.Entries[0].RemittanceInfo; got != "Rechnung RE-2026-0001 Muster GmbH" {
		t.Fatalf("remittance = %q", got)
	}
}

func TestDetectFormatIgnoresBOM(t *testing.T) {
	if got := DetectFormat([]byte("\ufeff<Document/>")); got != BankFormatCAMT053 {
		t.Fatalf("format = %q, want camt053 despite the BOM", got)
	}
}

func TestParseMT940WrappedInformationField(t *testing.T) {
	// German banks wrap :86: onto a second physical line without repeating the
	// tag; a continuation line belongs to the field still open, mid-token.
	content := ":20:WRAP\r\n:60F:C260101EUR0,00\r\n" +
		":61:260116CR119,00NTRFNONREF\r\n" +
		":86:?00GUTSCHRIFT?20Zahlung Rechn\r\n" +
		"ung RE-2026-0001?32Muster GmbH\r\n"
	stmt, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmt.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(stmt.Entries))
	}
	if got := stmt.Entries[0].RemittanceInfo; got != "Zahlung Rechnung RE-2026-0001" {
		t.Fatalf("remittance = %q, want the wrapped line joined mid-token", got)
	}
	if got := stmt.Entries[0].CounterpartyName; got != "Muster GmbH" {
		t.Fatalf("counterparty = %q", got)
	}
}

func TestParseMT940RejectsUnparseableBookingDateWithoutFailingTheEntry(t *testing.T) {
	// MMDD carries no year and no calendar validation of its own; a value that is
	// not a real date (Feb 31) must not fail the whole import \u2014 it silently
	// leaves BookingDate unset rather than guessing.
	content := ":20:BADBOOK\r\n:60F:C260101EUR0,00\r\n:61:2601160231CR10,00NTRFNONREF\r\n"
	stmt, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(stmt.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(stmt.Entries))
	}
	if stmt.Entries[0].BookingDate != nil {
		t.Fatalf("booking date = %v, want nil for an unparseable MMDD", stmt.Entries[0].BookingDate)
	}
	if !stmt.Entries[0].ValueDate.Equal(time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("value date = %v, must still parse independently of the bad booking date", stmt.Entries[0].ValueDate)
	}
}

func TestParseRejectsTooManyEntries(t *testing.T) {
	var b strings.Builder
	b.WriteString(":20:BULK\r\n:60F:C260101EUR0,00\r\n")
	for range maxEntriesPerStatement + 1 {
		b.WriteString(":61:260101CR1,00NTRFNONREF\r\n")
	}
	if _, err := Parse([]byte(b.String())); !errors.Is(err, ErrTooManyEntries) {
		t.Fatalf("err = %v, want ErrTooManyEntries", err)
	}
}

const camtMultiRemittanceSample = `<?xml version="1.0" encoding="UTF-8"?>
<Document xmlns="urn:iso:std:iso:20022:tech:xsd:camt.053.001.02">
  <BkToCstmrStmt>
    <GrpHdr><MsgId>MSG-2</MsgId><CreDtTm>2026-02-01T06:00:00</CreDtTm></GrpHdr>
    <Stmt>
      <Id>STMT-2026-02</Id>
      <Acct><Id><IBAN>DE02120300000000202051</IBAN></Id><Ccy>EUR</Ccy></Acct>
      <Ntry>
        <NtryRef>ENTRY-1</NtryRef>
        <Amt Ccy="EUR">250.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <Sts>BOOK</Sts>
        <ValDt><Dt>2026-01-15</Dt></ValDt>
        <NtryDtls><TxDtls>
          <RltdPties><Dbtr><Nm>Muster GmbH</Nm></Dbtr></RltdPties>
          <RmtInf><Ustrd>Sammelzahlung RE-1</Ustrd><Ustrd>und RE-2</Ustrd></RmtInf>
        </TxDtls></NtryDtls>
      </Ntry>
      <Ntry>
        <NtryRef>ENTRY-2</NtryRef>
        <Amt Ccy="EUR">30.00</Amt>
        <CdtDbtInd>CRDT</CdtDbtInd>
        <Sts>BOOK</Sts>
        <BookgDt><Dt>2026-01-20</Dt></BookgDt>
        <NtryDtls><TxDtls>
          <RltdPties><Dbtr><Nm>Ohne Valuta</Nm></Dbtr></RltdPties>
        </TxDtls></NtryDtls>
      </Ntry>
    </Stmt>
  </BkToCstmrStmt>
</Document>`

func TestParseCAMT053JoinsMultipleUnstructuredRemittanceLines(t *testing.T) {
	stmt, err := Parse([]byte(camtMultiRemittanceSample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := stmt.Entries[0].RemittanceInfo; got != "Sammelzahlung RE-1 und RE-2" {
		t.Fatalf("remittance = %q, want both Ustrd lines joined with a space", got)
	}
}

func TestParseCAMT053FallsBackToBookingDateWhenValueDateMissing(t *testing.T) {
	// Some banks omit ValDt on fee-style entries; reconciliation still needs a
	// date to work with, so BookgDt stands in rather than failing the entry.
	stmt, err := Parse([]byte(camtMultiRemittanceSample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	entry := stmt.Entries[1]
	want := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	if !entry.ValueDate.Equal(want) {
		t.Fatalf("value date = %v, want the booking date as fallback %v", entry.ValueDate, want)
	}
	if entry.BookingDate == nil || !entry.BookingDate.Equal(want) {
		t.Fatalf("booking date = %v, want %v", entry.BookingDate, want)
	}
}

func TestParseCAMT053RejectsEntryWithoutAnyUsableDate(t *testing.T) {
	sample := strings.Replace(camtMultiRemittanceSample,
		"<BookgDt><Dt>2026-01-20</Dt></BookgDt>", "", 1)
	if _, err := Parse([]byte(sample)); !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed for an entry with neither ValDt nor BookgDt", err)
	}
}

func TestParseCAMT053RejectsEntryWithEmptyAmount(t *testing.T) {
	sample := strings.Replace(camtMultiRemittanceSample,
		`<Amt Ccy="EUR">250.00</Amt>`, `<Amt Ccy="EUR"></Amt>`, 1)
	if _, err := Parse([]byte(sample)); !errors.Is(err, ErrMalformed) {
		t.Fatalf("err = %v, want ErrMalformed for an entry without an amount", err)
	}
}
