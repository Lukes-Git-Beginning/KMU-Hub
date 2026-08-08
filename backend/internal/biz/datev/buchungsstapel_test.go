package datev

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This file covers what upload_service_test.go's Builder tests (pagination,
// document counting, inverted period) do not: the ErrBuilderNotConfigured
// wiring guard and error propagation from the readers. Reuses the existing
// invoicePagerStub/creditNotePagerStub/settingsStub/periodBuilder fixtures
// from upload_service_test.go instead of building parallel fakes.

func TestBuild_NilBuilderReturnsErrBuilderNotConfigured(t *testing.T) {
	var b *BuchungsstapelBuilder
	_, err := b.Build(context.Background(), uuid.New(), time.Now(), time.Now(), time.Now())
	if !errors.Is(err, ErrBuilderNotConfigured) {
		t.Fatalf("expected ErrBuilderNotConfigured, got %v", err)
	}
}

func TestBuild_MissingInvoiceReaderReturnsErrBuilderNotConfigured(t *testing.T) {
	b := NewBuchungsstapelBuilder(NewExporter(), nil, &creditNotePagerStub{}, nil)
	_, err := b.Build(context.Background(), uuid.New(), time.Now(), time.Now(), time.Now())
	if !errors.Is(err, ErrBuilderNotConfigured) {
		t.Fatalf("expected ErrBuilderNotConfigured for a missing invoice reader, got %v", err)
	}
}

func TestBuild_MissingCreditNoteReaderReturnsErrBuilderNotConfigured(t *testing.T) {
	b := NewBuchungsstapelBuilder(NewExporter(), &invoicePagerStub{}, nil, nil)
	_, err := b.Build(context.Background(), uuid.New(), time.Now(), time.Now(), time.Now())
	if !errors.Is(err, ErrBuilderNotConfigured) {
		t.Fatalf("expected ErrBuilderNotConfigured for a missing credit note reader, got %v", err)
	}
}

// TestBuild_SettingsErrorIsBestEffortTolerated proves a failing
// CompanySettingsReader does not fail the export — Berater-/Mandantennummer
// stay empty and the caller decides whether that is acceptable (the GoBD
// download tolerates it, the DATEV API upload refuses per the doc comment on
// BuchungsstapelBuilder.BeraterNr/MandantNr).
func TestBuild_SettingsErrorIsBestEffortTolerated(t *testing.T) {
	b := NewBuchungsstapelBuilder(
		NewExporter(),
		&invoicePagerStub{},
		&creditNotePagerStub{},
		&settingsStub{err: errors.New("settings lookup failed")},
	)
	result, err := b.Build(context.Background(), uuid.New(), time.Now(), time.Now().Add(time.Hour), time.Now())
	if err != nil {
		t.Fatalf("Build should tolerate a settings error, got: %v", err)
	}
	if result.BeraterNr != "" || result.MandantNr != "" {
		t.Errorf("expected empty Berater/Mandant numbers on settings failure, got %q/%q", result.BeraterNr, result.MandantNr)
	}
}

func TestBuild_InvoiceReaderErrorPropagates(t *testing.T) {
	wantErr := errors.New("db unreachable")
	b := periodBuilder(nil, configuredSettings())
	b.invoices = &invoicePagerStub{err: wantErr}
	_, err := b.Build(context.Background(), uuid.New(), time.Now(), time.Now().Add(time.Hour), time.Now())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected the invoice reader error to propagate, got %v", err)
	}
}

func TestBuild_CreditNoteReaderErrorPropagates(t *testing.T) {
	wantErr := errors.New("db unreachable")
	b := periodBuilder(nil, configuredSettings())
	b.creditNotes = &creditNotePagerStub{err: wantErr}
	_, err := b.Build(context.Background(), uuid.New(), time.Now(), time.Now().Add(time.Hour), time.Now())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected the credit note reader error to propagate, got %v", err)
	}
}
