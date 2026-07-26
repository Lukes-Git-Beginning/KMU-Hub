package banking

// CAMT.053 (ISO 20022 bank-to-customer statement).
//
// The struct tags carry no namespace, so the same mapping reads camt.053.001.02
// through .08: encoding/xml then matches on the local element name alone, and
// the elements this importer needs have kept their names across those versions.
//
// encoding/xml resolves no external entities and expands no custom ones — an
// upload carrying a DTD entity bomb fails to decode rather than expanding, so no
// extra hardening is needed beyond the size limit in Parse.

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type camtDocument struct {
	XMLName   xml.Name        `xml:"Document"`
	Statement []camtStatement `xml:"BkToCstmrStmt>Stmt"`
}

type camtStatement struct {
	ID      string        `xml:"Id"`
	CreDtTm string        `xml:"CreDtTm"`
	Account camtAccount   `xml:"Acct"`
	Balance []camtBalance `xml:"Bal"`
	Entries []camtEntry   `xml:"Ntry"`
}

type camtAccount struct {
	IBAN     string `xml:"Id>IBAN"`
	Other    string `xml:"Id>Othr>Id"`
	Currency string `xml:"Ccy"`
}

type camtBalance struct {
	Code           string     `xml:"Tp>CdOrPrtry>Cd"`
	Amount         camtAmount `xml:"Amt"`
	CreditDebitInd string     `xml:"CdtDbtInd"`
	Date           camtDate   `xml:"Dt"`
}

type camtAmount struct {
	Currency string `xml:"Ccy,attr"`
	Value    string `xml:",chardata"`
}

// camtDate covers both <Dt><Dt>2026-01-05</Dt></Dt> (date) and
// <Dt><DtTm>2026-01-05T12:00:00</DtTm></Dt> (timestamp) shapes.
type camtDate struct {
	Date     string `xml:"Dt"`
	DateTime string `xml:"DtTm"`
}

type camtEntry struct {
	Ref            string          `xml:"NtryRef"`
	AcctSvcrRef    string          `xml:"AcctSvcrRef"`
	Amount         camtAmount      `xml:"Amt"`
	CreditDebitInd string          `xml:"CdtDbtInd"`
	Status         string          `xml:"Sts"`
	BookingDate    camtDate        `xml:"BookgDt"`
	ValueDate      camtDate        `xml:"ValDt"`
	AddtlNtryInf   string          `xml:"AddtlNtryInf"`
	Details        []camtTxDetails `xml:"NtryDtls>TxDtls"`
}

type camtTxDetails struct {
	EndToEndID   string      `xml:"Refs>EndToEndId"`
	Parties      camtParties `xml:"RltdPties"`
	Unstructured []string    `xml:"RmtInf>Ustrd"`
	// Structured creditor reference (RF…), used by some banks instead of Ustrd.
	CreditorRef string `xml:"RmtInf>Strd>CdtrRefInf>Ref"`
}

type camtParties struct {
	DebtorName   string `xml:"Dbtr>Nm"`
	DebtorIBAN   string `xml:"DbtrAcct>Id>IBAN"`
	CreditorName string `xml:"Cdtr>Nm"`
	CreditorIBAN string `xml:"CdtrAcct>Id>IBAN"`
}

// parseCAMT053 maps one CAMT.053 document to a ParsedStatement. A document may
// carry several <Stmt> blocks; only the first is imported, and that is a
// deliberate cut — see the lean note below.
func parseCAMT053(content []byte) (*ParsedStatement, error) {
	var doc camtDocument
	dec := xml.NewDecoder(strings.NewReader(string(content)))
	// Strict decoding: a well-formedness defect must surface as an error, never
	// as a silently truncated set of entries.
	dec.Strict = true
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if len(doc.Statement) == 0 {
		return nil, fmt.Errorf("%w: no BkToCstmrStmt/Stmt element", ErrMalformed)
	}

	// lean: only the first <Stmt> block is imported. Bank exports carry one
	// statement per file in practice; add a loop when a customer reports a
	// multi-statement export being cut short.
	src := doc.Statement[0]

	out := &ParsedStatement{
		Format:       BankFormatCAMT053,
		AccountIBAN:  strings.TrimSpace(firstNonEmpty(src.Account.IBAN, src.Account.Other)),
		StatementRef: strings.TrimSpace(src.ID),
		Currency:     strings.ToUpper(strings.TrimSpace(src.Account.Currency)),
	}
	if d, ok := parseCamtDate(src.Statement()); ok {
		out.StatementDate = &d
	}

	for _, bal := range src.Balance {
		amt, err := camtSignedAmount(bal.Amount, bal.CreditDebitInd)
		if err != nil {
			continue // a balance this importer cannot read is not worth failing the file over
		}
		switch strings.ToUpper(strings.TrimSpace(bal.Code)) {
		case "OPBD", "PRCD":
			v := amt
			out.OpeningBalance = &v
		case "CLBD":
			v := amt
			out.ClosingBalance = &v
		}
		if out.Currency == "" {
			out.Currency = strings.ToUpper(strings.TrimSpace(bal.Amount.Currency))
		}
	}

	for i := range src.Entries {
		entry, err := camtEntryToParsed(&src.Entries[i])
		if err != nil {
			return nil, err
		}
		if entry == nil {
			continue
		}
		out.Entries = append(out.Entries, *entry)
	}
	if out.Currency == "" && len(out.Entries) > 0 {
		out.Currency = out.Entries[0].Currency
	}
	return out, nil
}

// Statement returns the date carried by the statement header, as a camtDate so
// the shared date parsing applies.
func (s camtStatement) Statement() camtDate {
	return camtDate{DateTime: s.CreDtTm}
}

// camtEntryToParsed maps one <Ntry>. A pending entry (Sts != BOOK) is skipped:
// it may still change amount or vanish, and reconciling against it would book a
// payment that never arrived.
func camtEntryToParsed(e *camtEntry) (*ParsedEntry, error) {
	if sts := strings.ToUpper(strings.TrimSpace(e.Status)); sts != "" && sts != "BOOK" && sts != "BOOKED" {
		return nil, nil
	}

	amount, err := camtSignedAmount(e.Amount, e.CreditDebitInd)
	if err != nil {
		return nil, err
	}

	valueDate, ok := parseCamtDate(e.ValueDate)
	if !ok {
		// Fall back to the booking date: some banks omit ValDt on fee entries.
		valueDate, ok = parseCamtDate(e.BookingDate)
		if !ok {
			return nil, fmt.Errorf("%w: entry without a usable date", ErrMalformed)
		}
	}

	out := &ParsedEntry{
		EntryRef:  strings.TrimSpace(firstNonEmpty(e.Ref, e.AcctSvcrRef)),
		ValueDate: valueDate,
		Amount:    amount,
		Currency:  strings.ToUpper(strings.TrimSpace(e.Amount.Currency)),
	}
	if bd, okBd := parseCamtDate(e.BookingDate); okBd {
		out.BookingDate = &bd
	}

	// Collect remittance text and counterparty from the transaction details.
	// A batch entry carries several <TxDtls>; their remittance texts are joined
	// so an invoice number in any of them can still be found.
	var remittance []string
	for _, d := range e.Details {
		if out.EndToEndID == "" && strings.TrimSpace(d.EndToEndID) != "" &&
			!strings.EqualFold(strings.TrimSpace(d.EndToEndID), "NOTPROVIDED") {
			out.EndToEndID = strings.TrimSpace(d.EndToEndID)
		}
		for _, u := range d.Unstructured {
			if v := normaliseSpace(u); v != "" {
				remittance = append(remittance, v)
			}
		}
		if v := normaliseSpace(d.CreditorRef); v != "" {
			remittance = append(remittance, v)
		}
		// The counterparty is the other side of the flow: on money in that is
		// the debtor, on money out the creditor.
		name, iban := d.Parties.CreditorName, d.Parties.CreditorIBAN
		if amount.IsPositive() {
			name, iban = d.Parties.DebtorName, d.Parties.DebtorIBAN
		}
		if out.CounterpartyName == "" {
			out.CounterpartyName = normaliseSpace(name)
		}
		if out.CounterpartyIBAN == "" {
			out.CounterpartyIBAN = strings.TrimSpace(iban)
		}
	}
	if len(remittance) == 0 {
		// AddtlNtryInf is the only remittance text some banks fill.
		if v := normaliseSpace(e.AddtlNtryInf); v != "" {
			remittance = append(remittance, v)
		}
	}
	out.RemittanceInfo = strings.Join(remittance, " ")
	return out, nil
}

// camtSignedAmount turns the CAMT amount-plus-indicator pair into a signed
// decimal. An unreadable amount is an error, never a zero: a transaction booked
// as 0.00 would silently distort every reconciliation built on it.
func camtSignedAmount(a camtAmount, indicator string) (decimal.Decimal, error) {
	raw := strings.TrimSpace(a.Value)
	if raw == "" {
		return decimal.Zero, fmt.Errorf("%w: entry without an amount", ErrMalformed)
	}
	amount, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%w: unreadable amount %q", ErrMalformed, raw)
	}
	amount = amount.Abs()
	switch strings.ToUpper(strings.TrimSpace(indicator)) {
	case "CRDT":
		return amount, nil
	case "DBIT":
		return amount.Neg(), nil
	default:
		return decimal.Zero, fmt.Errorf("%w: missing or unknown CdtDbtInd %q", ErrMalformed, indicator)
	}
}

// parseCamtDate reads either the date or the timestamp variant, in UTC.
func parseCamtDate(d camtDate) (time.Time, bool) {
	if raw := strings.TrimSpace(d.Date); raw != "" {
		if t, err := time.Parse("2006-01-02", raw); err == nil {
			return t.UTC(), true
		}
	}
	raw := strings.TrimSpace(d.DateTime)
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Truncate(24 * time.Hour), true
		}
	}
	return time.Time{}, false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
