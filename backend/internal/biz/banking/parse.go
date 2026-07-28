// Package banking imports bank account statements (CAMT.053 and MT940) and
// reconciles their credit entries against open receivables.
//
// Everything a bank hands over is untrusted input: the parsers here never panic
// on malformed content, they return an error the transport turns into a 4xx.
// Structural strictness is deliberate — a statement that parses "somehow" would
// produce plausible-looking transactions with wrong amounts, which is worse
// than a rejected upload.
package banking

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// MaxStatementBytes bounds an accepted statement file (10 MiB). A CAMT.053 for a
// full month of a busy account stays well under a megabyte; the limit exists to
// stop a decompression- or allocation-heavy upload, not to constrain real use.
const MaxStatementBytes = 10 << 20

// maxEntriesPerStatement bounds how many entries one file may carry. A file
// beyond this is a bulk export or an attack, not a monthly statement.
const maxEntriesPerStatement = 20000

// ParsedStatement is one statement file, format-independent.
type ParsedStatement struct {
	Format         string
	AccountIBAN    string
	StatementRef   string
	Currency       string
	OpeningBalance *decimal.Decimal
	ClosingBalance *decimal.Decimal
	StatementDate  *time.Time
	Entries        []ParsedEntry
}

// ParsedEntry is one booked line of a statement. Amount is signed: a credit is
// positive, a debit negative, so the sign survives into the database and every
// aggregate stays a plain sum.
type ParsedEntry struct {
	EntryRef         string
	EndToEndID       string
	ValueDate        time.Time
	BookingDate      *time.Time
	Amount           decimal.Decimal
	Currency         string
	CounterpartyName string
	CounterpartyIBAN string
	RemittanceInfo   string
}

// Parse errors. All of them are client faults — the transport maps them to 4xx.
var (
	// ErrEmptyFile is returned for an upload with no content.
	ErrEmptyFile = errors.New("banking: uploaded file is empty")
	// ErrFileTooLarge is returned above MaxStatementBytes.
	ErrFileTooLarge = errors.New("banking: file exceeds the size limit")
	// ErrUnknownFormat is returned when the content resembles neither format.
	ErrUnknownFormat = errors.New("banking: unrecognised statement format, expected CAMT.053 XML or MT940")
	// ErrMalformed wraps a format-specific structural defect.
	ErrMalformed = errors.New("banking: malformed statement")
	// ErrTooManyEntries is returned above maxEntriesPerStatement.
	ErrTooManyEntries = errors.New("banking: statement carries too many entries")
)

// DetectFormat classifies a statement file by its content, not by its filename
// or MIME type: both are attacker- and browser-controlled, the content is what
// gets parsed.
func DetectFormat(content []byte) string {
	// \ufeff strips a UTF-8 BOM: exports written on Windows carry one, and it
	// would otherwise hide the leading "<" of an XML document.
	head := strings.TrimLeft(string(content[:min(len(content), 4096)]), " \t\r\n\ufeff")
	switch {
	case strings.HasPrefix(head, "<"):
		return BankFormatCAMT053
	case strings.Contains(head, ":20:"):
		// Every MT940 message opens with the transaction reference field.
		return BankFormatMT940
	default:
		return ""
	}
}

// Statement file formats. Mirrors models.BankStatementFormat* — duplicated here
// so the parser package does not depend on the persistence models.
const (
	// BankFormatCAMT053 is the ISO 20022 XML account statement.
	BankFormatCAMT053 = "camt053"
	// BankFormatMT940 is the SWIFT text account statement.
	BankFormatMT940 = "mt940"
)

// Parse detects the format and parses the file. It is the only entry point the
// service uses; callers never pick a parser themselves, so an unknown format can
// never fall through to a lenient one.
func Parse(content []byte) (*ParsedStatement, error) {
	if len(content) == 0 {
		return nil, ErrEmptyFile
	}
	if len(content) > MaxStatementBytes {
		return nil, ErrFileTooLarge
	}

	var (
		stmt *ParsedStatement
		err  error
	)
	switch DetectFormat(content) {
	case BankFormatCAMT053:
		stmt, err = parseCAMT053(content)
	case BankFormatMT940:
		stmt, err = parseMT940(content)
	default:
		return nil, ErrUnknownFormat
	}
	if err != nil {
		return nil, err
	}
	if len(stmt.Entries) > maxEntriesPerStatement {
		return nil, ErrTooManyEntries
	}
	return stmt, nil
}

// normaliseSpace collapses runs of whitespace into single spaces and trims. Bank
// remittance text arrives wrapped across fixed-width lines; matching an invoice
// number against it only works once the wrapping is gone.
func normaliseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
