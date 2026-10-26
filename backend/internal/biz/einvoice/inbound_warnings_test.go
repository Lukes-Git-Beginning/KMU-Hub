package einvoice

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return string(b)
}

// ciiWith returns the CII fixture with a currency and a header VAT category code.
func ciiWith(t *testing.T, currency, category string) []byte {
	x := readFixture(t, "cii_minimal.xml")
	x = strings.ReplaceAll(x, "<ram:InvoiceCurrencyCode>EUR<", "<ram:InvoiceCurrencyCode>"+currency+"<")
	x = strings.ReplaceAll(x, "<ram:TypeCode>VAT</ram:TypeCode>", "<ram:TypeCode>VAT</ram:TypeCode><ram:CategoryCode>"+category+"</ram:CategoryCode>")
	return []byte(x)
}

func TestCheckCodelists(t *testing.T) {
	tests := []struct {
		name      string
		parsed    ParsedInvoice
		wantRules []string
	}{
		{"supported currency and category", ParsedInvoice{Currency: "EUR", TaxCategories: []string{"S", "AE"}}, nil},
		{"lower-case currency is normalised", ParsedInvoice{Currency: " chf "}, nil},
		{"no category stated", ParsedInvoice{Currency: "USD"}, nil},
		{"unknown currency", ParsedInvoice{Currency: "GBP", TaxCategories: []string{"S"}}, []string{"BR-CL-04"}},
		{"unknown category", ParsedInvoice{Currency: "EUR", TaxCategories: []string{"S", "K"}}, []string{"BR-CL-17"}},
		{"both, one warning per category", ParsedInvoice{Currency: "XXX", TaxCategories: []string{"K", "O"}}, []string{"BR-CL-04", "BR-CL-17", "BR-CL-17"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, w := range CheckCodelists(&tt.parsed) {
				got = append(got, w.Rule)
				assert.NotEmpty(t, w.Term)
				assert.NotEmpty(t, w.Message)
			}
			assert.Equal(t, tt.wantRules, got)
		})
	}
}

func TestDistinctCategories(t *testing.T) {
	assert.Equal(t, []string{"S", "AE"}, distinctCategories([]string{"s", " S ", "", "ae", "AE"}))
	assert.Empty(t, distinctCategories(nil))
}

func TestParseCII_ReadsHeaderTaxCategories(t *testing.T) {
	parsed, err := ParseCII(ciiWith(t, "EUR", "k"))
	require.NoError(t, err)
	assert.Equal(t, []string{"K"}, parsed.TaxCategories)
}

func TestParseUBL_ReadsHeaderTaxCategories(t *testing.T) {
	parsed, err := ParseUBL([]byte(readFixture(t, "ubl_minimal.xml")))
	require.NoError(t, err)
	assert.Equal(t, []string{"S"}, parsed.TaxCategories, "line-level ClassifiedTaxCategory must not add a second S")
}

func TestImportWithWarnings_StoresInvoiceAndReportsDeviations(t *testing.T) {
	repo := newMockRepo()
	svc := NewService(repo)
	tenant := uuid.New()

	inv, warnings, err := svc.ImportWithWarnings(context.Background(), ImportInput{
		TenantID:  tenant,
		CreatedBy: uuid.New(),
		Filename:  "gbp.xml",
		Content:   ciiWith(t, "GBP", "K"),
		MimeType:  "application/xml",
	})
	require.NoError(t, err, "a codelist deviation must not reject the invoice")
	require.NotNil(t, inv)
	assert.Equal(t, "GBP", inv.Currency)
	assert.Contains(t, repo.invoices, inv.ID, "the invoice is stored despite the warnings")

	var rules []string
	for _, w := range warnings {
		rules = append(rules, w.Rule)
	}
	assert.Equal(t, []string{"BR-CL-04", "BR-CL-17"}, rules)
}

func TestImportWithWarnings_CleanInvoiceHasNoWarnings(t *testing.T) {
	svc := NewService(newMockRepo())
	_, warnings, err := svc.ImportWithWarnings(context.Background(), ImportInput{
		TenantID:  uuid.New(),
		CreatedBy: uuid.New(),
		Filename:  "ok.xml",
		Content:   ciiWith(t, "EUR", "S"),
		MimeType:  "application/xml",
	})
	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestImportWithWarnings_ErrorReturnsNoWarnings(t *testing.T) {
	_, warnings, err := NewService(newMockRepo()).ImportWithWarnings(context.Background(), ImportInput{
		TenantID: uuid.New(), Content: []byte("not xml"), MimeType: "application/xml",
	})
	require.Error(t, err)
	assert.Nil(t, warnings)
}
