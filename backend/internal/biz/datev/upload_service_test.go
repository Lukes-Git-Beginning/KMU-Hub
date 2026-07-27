package datev

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Fakes
// ============================================================================

// invoicePagerStub serves pre-built pages and records the tenant and cursor it
// was asked for, so the paging loop can be pinned without a database.
type invoicePagerStub struct {
	pages      [][]*models.Invoice
	callTenant []uuid.UUID
	callCursor []*uuid.UUID
	err        error
}

func (s *invoicePagerStub) ListForDATEVExport(_ context.Context, tenantID uuid.UUID, _, _ time.Time,
	_ *time.Time, afterID *uuid.UUID, _ int) ([]*models.Invoice, error) {
	if s.err != nil {
		return nil, s.err
	}
	idx := len(s.callTenant)
	s.callTenant = append(s.callTenant, tenantID)
	s.callCursor = append(s.callCursor, afterID)
	if idx >= len(s.pages) {
		return nil, nil
	}
	return s.pages[idx], nil
}

type creditNotePagerStub struct {
	pages      [][]*models.CreditNote
	callTenant []uuid.UUID
}

func (s *creditNotePagerStub) ListForDATEVExport(_ context.Context, tenantID uuid.UUID, _, _ time.Time,
	_ *time.Time, _ *uuid.UUID, _ int) ([]*models.CreditNote, error) {
	idx := len(s.callTenant)
	s.callTenant = append(s.callTenant, tenantID)
	if idx >= len(s.pages) {
		return nil, nil
	}
	return s.pages[idx], nil
}

type settingsStub struct {
	settings *models.CompanySettings
	err      error
}

func (s *settingsStub) GetByTenantID(_ context.Context, _ uuid.UUID) (*models.CompanySettings, error) {
	return s.settings, s.err
}

type configRepoStub struct {
	config *IntegrationConfig
	err    error
}

func (s *configRepoStub) GetByPlatform(_ context.Context, _ string) (*IntegrationConfig, error) {
	return s.config, s.err
}
func (s *configRepoStub) Upsert(_ context.Context, _ *IntegrationConfig) error { return nil }
func (s *configRepoStub) Deactivate(_ context.Context, _ uuid.UUID) error      { return nil }

type uploadRepoStub struct {
	uploadConfig *models.DatevUploadConfig
	configErr    error
	logs         []*models.DatevUploadLog
}

func (s *uploadRepoStub) GetUploadConfig(_ context.Context, _ uuid.UUID) (*models.DatevUploadConfig, error) {
	return s.uploadConfig, s.configErr
}
func (s *uploadRepoStub) UpsertUploadConfig(_ context.Context, _ *models.DatevUploadConfig) error {
	return nil
}
func (s *uploadRepoStub) CreateUploadLog(_ context.Context, l *models.DatevUploadLog) error {
	s.logs = append(s.logs, l)
	return nil
}
func (s *uploadRepoStub) UpdateUploadLog(_ context.Context, _ *models.DatevUploadLog) error {
	return nil
}
func (s *uploadRepoStub) ListUploadLogs(_ context.Context, _ uuid.UUID, _ int) ([]models.DatevUploadLog, error) {
	return nil, nil
}

// uploaderSpy stands in for the DATEV API. It counts calls, because the defect
// this endpoint had was reporting success without one.
type uploaderSpy struct {
	calls        int
	clientNumber string
	csv          []byte
	err          error
}

func (s *uploaderSpy) UploadBuchungsstapel(_ context.Context, _ uuid.UUID, clientNumber string, csvData []byte) error {
	s.calls++
	s.clientNumber = clientNumber
	s.csv = csvData
	return s.err
}

type belegUploaderSpy struct {
	calls    int
	pdf      []byte
	filename string
	err      error
}

func (s *belegUploaderSpy) UploadBeleg(_ context.Context, _ uuid.UUID, _ string, pdfData []byte, filename string) error {
	s.calls++
	s.pdf = pdfData
	s.filename = filename
	return s.err
}

type belegSourceStub struct {
	pdf      []byte
	filename string
	err      error
	calls    int
}

func (s *belegSourceStub) RenderInvoice(_ context.Context, _, _ uuid.UUID) ([]byte, string, error) {
	s.calls++
	return s.pdf, s.filename, s.err
}

// ============================================================================
// Fixtures
// ============================================================================

func sentInvoice(number string) *models.Invoice {
	return &models.Invoice{
		ID:            uuid.New(),
		InvoiceNumber: number,
		Status:        models.InvoiceStatusSent,
		CustomerName:  "Test GmbH",
		TaxMode:       models.TaxModeStandard,
		Currency:      "EUR",
		InvoiceDate:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		LineItems: makeLineItems([]models.LineItem{{
			ID:          "1",
			Position:    1,
			Description: "Beratung",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.NewFromFloat(100),
			TaxRate:     decimal.NewFromInt(19),
			LineTotal:   decimal.NewFromFloat(100),
		}}),
	}
}

func draftInvoice(number string) *models.Invoice {
	inv := sentInvoice(number)
	inv.Status = models.InvoiceStatusDraft
	return inv
}

func configuredSettings() *models.CompanySettings {
	return &models.CompanySettings{DatevBeraterNr: "12345", DatevMandantNr: "67890"}
}

// connectedService builds an upload service whose preconditions are all met, so
// each test can knock out exactly one of them.
func connectedService(t *testing.T, builder *BuchungsstapelBuilder, up *uploaderSpy) (*UploadService, *uploadRepoStub) {
	t.Helper()
	repo := &uploadRepoStub{uploadConfig: &models.DatevUploadConfig{ClientNumber: "55555"}}
	svc := NewUploadService(
		builder,
		&belegSourceStub{pdf: []byte("%PDF-1.4"), filename: "Rechnung_RE-1.pdf"},
		up,
		&belegUploaderSpy{},
		repo,
		&configRepoStub{config: &IntegrationConfig{ID: uuid.New(), IsActive: true}},
		&OAuthManager{},
	)
	return svc, repo
}

func periodBuilder(inv []*models.Invoice, settings *models.CompanySettings) *BuchungsstapelBuilder {
	return NewBuchungsstapelBuilder(
		NewExporter(),
		&invoicePagerStub{pages: [][]*models.Invoice{inv}},
		&creditNotePagerStub{},
		&settingsStub{settings: settings},
	)
}

// ============================================================================
// Builder
// ============================================================================

func TestBuchungsstapelBuilder_CountsOnlyExportedDocuments(t *testing.T) {
	// A draft carries no booking line. Counting it would tell the client that a
	// document reached DATEV which the file never contained.
	builder := periodBuilder([]*models.Invoice{sentInvoice("RE-1"), draftInvoice("RE-2")}, configuredSettings())

	batch, err := builder.Build(context.Background(), uuid.New(),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if batch.DocumentCount != 1 {
		t.Errorf("DocumentCount = %d, want 1 (the draft must not count)", batch.DocumentCount)
	}
	if batch.LineCount != 1 {
		t.Errorf("LineCount = %d, want 1", batch.LineCount)
	}
	if batch.BeraterNr != "12345" || batch.MandantNr != "67890" {
		t.Errorf("advisor numbers = %q/%q, want 12345/67890", batch.BeraterNr, batch.MandantNr)
	}
	if !strings.Contains(string(batch.CSV), "12345") {
		t.Error("Beraternummer missing from the EXTF header")
	}
}

func TestBuchungsstapelBuilder_PagesWithKeysetCursor(t *testing.T) {
	// A full page must be followed by another read carrying the last row's id as
	// the cursor — a builder that forgets the cursor loops forever on page one.
	full := make([]*models.Invoice, buchungsstapelPageSize)
	for i := range full {
		full[i] = sentInvoice("RE-full")
	}
	tail := []*models.Invoice{sentInvoice("RE-tail")}
	pager := &invoicePagerStub{pages: [][]*models.Invoice{full, tail}}

	builder := NewBuchungsstapelBuilder(NewExporter(), pager, &creditNotePagerStub{}, &settingsStub{settings: configuredSettings()})
	tenantID := uuid.New()

	batch, err := builder.Build(context.Background(), tenantID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(pager.callCursor) != 2 {
		t.Fatalf("pager called %d times, want 2", len(pager.callCursor))
	}
	if pager.callCursor[0] != nil {
		t.Error("first read must start without a cursor")
	}
	if pager.callCursor[1] == nil || *pager.callCursor[1] != full[len(full)-1].ID {
		t.Error("second read must continue after the last row of page one")
	}
	for _, got := range pager.callTenant {
		if got != tenantID {
			t.Errorf("read scoped to tenant %s, want %s", got, tenantID)
		}
	}
	if batch.DocumentCount != buchungsstapelPageSize+1 {
		t.Errorf("DocumentCount = %d, want %d", batch.DocumentCount, buchungsstapelPageSize+1)
	}
}

func TestBuchungsstapelBuilder_RejectsInvertedPeriod(t *testing.T) {
	builder := periodBuilder(nil, configuredSettings())

	_, err := builder.Build(context.Background(), uuid.New(),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if !errors.Is(err, ErrInvalidPeriod) {
		t.Fatalf("err = %v, want ErrInvalidPeriod", err)
	}
}

// ============================================================================
// Upload — no success without a transfer
// ============================================================================

func TestUploadBuchungsstapel_RefusesWithoutTransfer(t *testing.T) {
	period := func(svc *UploadService) error {
		_, err := svc.UploadBuchungsstapel(context.Background(), uuid.New(),
			time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		return err
	}

	tests := []struct {
		name    string
		mutate  func(svc *UploadService)
		builder *BuchungsstapelBuilder
		want    error
	}{
		{
			name:    "no api credentials wired",
			mutate:  func(svc *UploadService) { svc.uploader = nil },
			builder: periodBuilder([]*models.Invoice{sentInvoice("RE-1")}, configuredSettings()),
			want:    ErrNotConnected,
		},
		{
			name: "tenant integration inactive",
			mutate: func(svc *UploadService) {
				svc.configRepo = &configRepoStub{config: &IntegrationConfig{ID: uuid.New(), IsActive: false}}
			},
			builder: periodBuilder([]*models.Invoice{sentInvoice("RE-1")}, configuredSettings()),
			want:    ErrNoAPIConfig,
		},
		{
			name: "client number unset",
			mutate: func(svc *UploadService) {
				svc.uploadRepo = &uploadRepoStub{uploadConfig: &models.DatevUploadConfig{ClientNumber: ""}}
			},
			builder: periodBuilder([]*models.Invoice{sentInvoice("RE-1")}, configuredSettings()),
			want:    ErrNoUploadConfig,
		},
		{
			name:    "advisor numbers unset",
			mutate:  func(_ *UploadService) {},
			builder: periodBuilder([]*models.Invoice{sentInvoice("RE-1")}, &models.CompanySettings{}),
			want:    ErrAdvisorNumbersMissing,
		},
		{
			name:    "period holds no bookable document",
			mutate:  func(_ *UploadService) {},
			builder: periodBuilder([]*models.Invoice{draftInvoice("RE-1")}, configuredSettings()),
			want:    ErrNothingToUpload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			up := &uploaderSpy{}
			svc, _ := connectedService(t, tt.builder, up)
			tt.mutate(svc)

			err := period(svc)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if up.calls != 0 {
				t.Errorf("platform contacted %d times, want 0", up.calls)
			}
		})
	}
}

func TestUploadBuchungsstapel_TransfersBatchAndLogsCompleted(t *testing.T) {
	up := &uploaderSpy{}
	svc, repo := connectedService(t, periodBuilder([]*models.Invoice{sentInvoice("RE-1")}, configuredSettings()), up)

	batch, err := svc.UploadBuchungsstapel(context.Background(), uuid.New(),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("UploadBuchungsstapel: %v", err)
	}

	if up.calls != 1 {
		t.Fatalf("platform contacted %d times, want 1", up.calls)
	}
	if up.clientNumber != "55555" {
		t.Errorf("client number = %q, want 55555", up.clientNumber)
	}
	if len(up.csv) == 0 || !strings.Contains(string(up.csv), "RE-1") {
		t.Error("the uploaded CSV does not carry the invoice")
	}
	if batch.DocumentCount != 1 {
		t.Errorf("DocumentCount = %d, want 1", batch.DocumentCount)
	}
	if len(repo.logs) != 1 {
		t.Fatalf("wrote %d upload logs, want 1", len(repo.logs))
	}
	if repo.logs[0].Status != "completed" {
		t.Errorf("log status = %q, want completed", repo.logs[0].Status)
	}
	if repo.logs[0].DocumentCount != 1 {
		t.Errorf("logged DocumentCount = %d, want 1", repo.logs[0].DocumentCount)
	}
}

func TestUploadBuchungsstapel_FailedTransferIsNoSuccess(t *testing.T) {
	up := &uploaderSpy{err: errors.New("502 from DATEV")}
	svc, repo := connectedService(t, periodBuilder([]*models.Invoice{sentInvoice("RE-1")}, configuredSettings()), up)

	batch, err := svc.UploadBuchungsstapel(context.Background(), uuid.New(),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected an error when the transfer fails")
	}
	if batch != nil {
		t.Error("a failed transfer must not return a batch")
	}
	if repo.logs[0].Status != "failed" {
		t.Errorf("log status = %q, want failed", repo.logs[0].Status)
	}
}

// ============================================================================
// Belegbild
// ============================================================================

func TestUploadInvoiceBeleg_TransfersRenderedPDF(t *testing.T) {
	belegUp := &belegUploaderSpy{}
	source := &belegSourceStub{pdf: []byte("%PDF-1.4 rendered"), filename: "Rechnung_RE-1.pdf"}
	svc := NewUploadService(
		periodBuilder(nil, configuredSettings()), source, &uploaderSpy{}, belegUp,
		&uploadRepoStub{uploadConfig: &models.DatevUploadConfig{ClientNumber: "55555"}},
		&configRepoStub{config: &IntegrationConfig{ID: uuid.New(), IsActive: true}},
		&OAuthManager{},
	)

	if err := svc.UploadInvoiceBeleg(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("UploadInvoiceBeleg: %v", err)
	}
	if belegUp.calls != 1 {
		t.Fatalf("platform contacted %d times, want 1", belegUp.calls)
	}
	if string(belegUp.pdf) != "%PDF-1.4 rendered" {
		t.Error("the rendered PDF is not what was transferred")
	}
	if belegUp.filename != "Rechnung_RE-1.pdf" {
		t.Errorf("filename = %q, want Rechnung_RE-1.pdf", belegUp.filename)
	}
}

func TestUploadInvoiceBeleg_RefusesWhenNothingRendered(t *testing.T) {
	tests := map[string]*belegSourceStub{
		"render fails":     {err: ErrInvoiceNotFound},
		"render is empty":  {pdf: nil, filename: "x.pdf"},
		"settings missing": {err: ErrCompanySettingsIncomplete},
	}

	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			belegUp := &belegUploaderSpy{}
			svc := NewUploadService(
				periodBuilder(nil, configuredSettings()), source, &uploaderSpy{}, belegUp,
				&uploadRepoStub{uploadConfig: &models.DatevUploadConfig{ClientNumber: "55555"}},
				&configRepoStub{config: &IntegrationConfig{ID: uuid.New(), IsActive: true}},
				&OAuthManager{},
			)

			if err := svc.UploadInvoiceBeleg(context.Background(), uuid.New(), uuid.New()); err == nil {
				t.Fatal("expected an error, got success")
			}
			if belegUp.calls != 0 {
				t.Errorf("platform contacted %d times, want 0", belegUp.calls)
			}
		})
	}
}
