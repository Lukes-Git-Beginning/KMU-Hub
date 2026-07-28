package banking

// MT940 (SWIFT customer statement message).
//
// The format is line-oriented: a tag ":NN:" opens a field, everything up to the
// next tag belongs to it. Two fields carry the payload — :61: the booking line
// (dates, sign, amount, references) and :86: the free text, which German banks
// subdivide further with "?NN" markers.
//
// A defect here is not cosmetic: misreading the sign or the decimal comma turns
// an incoming payment into an outgoing one, or 1.234,56 into 1.23. Both fields
// are therefore parsed strictly, and an unreadable :61: fails the import.

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// mt940Statement line prefixes.
const (
	tagTransactionRef = ":20:"
	tagAccountID      = ":25:"
	tagOpeningBalance = ":60F:"
	tagOpeningInterim = ":60M:"
	tagStatementLine  = ":61:"
	tagInformation    = ":86:"
	tagClosingBalance = ":62F:"
	tagClosingInterim = ":62M:"
)

// mt940LineRE splits a :61: statement line.
//
//	1 value date         YYMMDD
//	2 booking date       MMDD, optional
//	3 debit/credit mark  C | D | RC | RD (R… = reversal)
//	4 funds code         one letter, optional (third currency character)
//	5 amount             digits with a comma as decimal separator
//	6 transaction type   one letter plus three alphanumerics, e.g. NTRF
//	7 references         customer reference, optionally //bank reference
var mt940LineRE = regexp.MustCompile(
	`^(\d{6})(\d{4})?(RC|RD|C|D)([A-Z])?([0-9]+(?:,[0-9]*)?)([A-Z][A-Z0-9]{3})?(.*)$`,
)

// mt940BalanceRE splits :60F:/:62F: — mark, date, currency, amount.
var mt940BalanceRE = regexp.MustCompile(`^([CD])(\d{6})([A-Z]{3})([0-9]+(?:,[0-9]*)?)$`)

// mt940SubfieldRE finds the "?NN" markers inside a :86: field.
var mt940SubfieldRE = regexp.MustCompile(`\?(\d{2})`)

func parseMT940(content []byte) (*ParsedStatement, error) {
	out := &ParsedStatement{Format: BankFormatMT940}

	fields := splitMT940Fields(string(content))
	if len(fields) == 0 {
		return nil, fmt.Errorf("%w: no MT940 fields found", ErrMalformed)
	}

	// pending is the entry opened by the last :61: and still waiting for its
	// :86: text. It is appended when the next :61: arrives or at the end, so an
	// entry is never dropped just because its information field is missing.
	var pending *ParsedEntry
	flush := func() {
		if pending != nil {
			out.Entries = append(out.Entries, *pending)
			pending = nil
		}
	}

	for _, f := range fields {
		switch f.tag {
		case tagTransactionRef:
			if out.StatementRef == "" {
				out.StatementRef = strings.TrimSpace(f.value)
			}
		case tagAccountID:
			if out.AccountIBAN == "" {
				// ":25:DE12…/EUR" — the account id may carry a currency suffix.
				out.AccountIBAN = strings.TrimSpace(strings.SplitN(f.value, "/", 2)[0])
			}
		case tagOpeningBalance, tagOpeningInterim:
			if bal, ccy, ok := parseMT940Balance(f.value); ok && out.OpeningBalance == nil {
				out.OpeningBalance = &bal
				if out.Currency == "" {
					out.Currency = ccy
				}
			}
		case tagClosingBalance, tagClosingInterim:
			if bal, ccy, ok := parseMT940Balance(f.value); ok {
				out.ClosingBalance = &bal
				if out.Currency == "" {
					out.Currency = ccy
				}
				if d, dok := parseMT940BalanceDate(f.value); dok {
					out.StatementDate = &d
				}
			}
		case tagStatementLine:
			flush()
			entry, err := parseMT940Line(f.value, out.Currency)
			if err != nil {
				return nil, err
			}
			pending = entry
		case tagInformation:
			if pending != nil {
				applyMT940Information(pending, f.value)
			}
		}
	}
	flush()

	// The currency is not repeated per entry in MT940; it comes from the
	// balance fields and applies to the whole statement.
	for i := range out.Entries {
		if out.Entries[i].Currency == "" {
			out.Entries[i].Currency = out.Currency
		}
	}
	return out, nil
}

// mt940Field is one tag with its (possibly multi-line) value.
type mt940Field struct {
	tag   string
	value string
}

// mt940TagRE matches a tag at the start of a line, e.g. ":61:" or ":60F:".
var mt940TagRE = regexp.MustCompile(`^:(\d{2}[A-Z]?):`)

// splitMT940Fields folds the physical lines into tag/value pairs. A continuation
// line (no tag) is appended to the field that is currently open.
func splitMT940Fields(content string) []mt940Field {
	var fields []mt940Field
	for _, raw := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" {
			continue
		}
		if m := mt940TagRE.FindString(line); m != "" {
			fields = append(fields, mt940Field{tag: m, value: line[len(m):]})
			continue
		}
		if len(fields) > 0 {
			fields[len(fields)-1].value += line
		}
	}
	return fields
}

// parseMT940Line reads one :61: booking line.
func parseMT940Line(value, fallbackCurrency string) (*ParsedEntry, error) {
	m := mt940LineRE.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return nil, fmt.Errorf("%w: unreadable :61: line %q", ErrMalformed, value)
	}

	valueDate, err := time.Parse("060102", m[1])
	if err != nil {
		return nil, fmt.Errorf("%w: unreadable value date in :61: %q", ErrMalformed, value)
	}
	valueDate = valueDate.UTC()

	amount, err := parseMT940Amount(m[5])
	if err != nil {
		return nil, fmt.Errorf("%w: unreadable amount in :61: %q", ErrMalformed, value)
	}
	// D and RC move money out, C and RD move it in: a reversal flips the sign of
	// the mark it reverses.
	switch m[3] {
	case "D", "RC":
		amount = amount.Neg()
	}

	entry := &ParsedEntry{
		ValueDate: valueDate,
		Amount:    amount,
		Currency:  fallbackCurrency,
	}

	// The booking date carries no year; it belongs to the value date's year
	// except across a turn of the year, where a December booking can be paired
	// with a January value date.
	if m[2] != "" {
		if bd, bok := parseMT940BookingDate(m[2], valueDate); bok {
			entry.BookingDate = &bd
		}
	}

	// References: "customer//bank". NONREF is the placeholder for "none given".
	refs := strings.TrimSpace(m[7])
	customerRef, bankRef, _ := strings.Cut(refs, "//")
	customerRef = strings.TrimSpace(customerRef)
	bankRef = strings.TrimSpace(bankRef)
	if strings.EqualFold(customerRef, "NONREF") {
		customerRef = ""
	}
	entry.EntryRef = firstNonEmpty(bankRef, customerRef)
	entry.EndToEndID = customerRef
	return entry, nil
}

// applyMT940Information reads a :86: field into the entry. German banks encode
// subfields as "?NN"; a field without them is taken as remittance text whole.
func applyMT940Information(entry *ParsedEntry, value string) {
	idx := mt940SubfieldRE.FindAllStringSubmatchIndex(value, -1)
	if len(idx) == 0 {
		entry.RemittanceInfo = normaliseSpace(value)
		return
	}

	var remittance, name []string
	for i, loc := range idx {
		code := value[loc[2]:loc[3]]
		end := len(value)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		text := strings.TrimSpace(value[loc[1]:end])
		if text == "" {
			continue
		}
		switch {
		// ?20–?29 and ?60–?63 are the remittance text, split across fixed-width
		// chunks. They are joined without a separator: a bank splits an invoice
		// number mid-token, and inserting a space there would break the match.
		case code >= "20" && code <= "29", code >= "60" && code <= "63":
			remittance = append(remittance, text)
		case code == "32", code == "33":
			name = append(name, text)
		case code == "31":
			entry.CounterpartyIBAN = strings.TrimSpace(text)
		}
	}
	entry.RemittanceInfo = normaliseSpace(strings.Join(remittance, ""))
	entry.CounterpartyName = normaliseSpace(strings.Join(name, ""))
}

// parseMT940Amount converts "1234,56" to a decimal. The comma is the decimal
// separator in this format; there is no thousands separator.
func parseMT940Amount(raw string) (decimal.Decimal, error) {
	return decimal.NewFromString(strings.Replace(strings.TrimSpace(raw), ",", ".", 1))
}

// parseMT940Balance reads a :60F:/:62F: field into a signed amount plus currency.
func parseMT940Balance(value string) (decimal.Decimal, string, bool) {
	m := mt940BalanceRE.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return decimal.Zero, "", false
	}
	amount, err := parseMT940Amount(m[4])
	if err != nil {
		return decimal.Zero, "", false
	}
	if m[1] == "D" {
		amount = amount.Neg()
	}
	return amount, m[3], true
}

// parseMT940BalanceDate reads the date out of a balance field.
func parseMT940BalanceDate(value string) (time.Time, bool) {
	m := mt940BalanceRE.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return time.Time{}, false
	}
	t, err := time.Parse("060102", m[2])
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// parseMT940BookingDate resolves the year-less MMDD booking date against the
// value date. A booking in December paired with a value date in January belongs
// to the previous year, and the mirror case to the next one — without this a
// year-end statement dates half its entries eleven months off.
func parseMT940BookingDate(mmdd string, valueDate time.Time) (time.Time, bool) {
	t, err := time.Parse("0102", mmdd)
	if err != nil {
		return time.Time{}, false
	}
	year := valueDate.Year()
	switch {
	case t.Month() == time.December && valueDate.Month() == time.January:
		year--
	case t.Month() == time.January && valueDate.Month() == time.December:
		year++
	}
	return time.Date(year, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
}
