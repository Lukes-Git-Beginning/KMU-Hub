package export_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/berichte"
	"github.com/kmuhub/kmuhub/internal/berichte/export"
)

const sampleDocumentRows = `[
  {"columns":[{"width":1,"blocks":[
    {"id":"b1","type":"cover","title":"Q3 Bericht","subtitle":"Vertrieb","author":"Nico"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b2","type":"heading","level":1,"text":"Kennzahlen"}
  ]}]},
  {"columns":[
    {"width":1,"blocks":[{"id":"k1","type":"kpi","label":"Umsatz","value":"12.345","unit":"EUR","changePercent":4.2}]},
    {"width":1,"blocks":[{"id":"k2","type":"kpi","label":"Neukunden","value":"18","changePercent":-2.5}]}
  ]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b3","type":"text","html":"<p>Umsatz <strong>steigt</strong> im dritten Quartal.</p>"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b4","type":"chart","definitionId":"11111111-1111-1111-1111-111111111111","caption":"Umsatz nach Monat"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b5","type":"table","caption":"Ohne gespeicherte Definition"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b6","type":"callout","variant":"warning","title":"Achtung","html":"<p>Bestand niedrig</p>"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b7","type":"bullet","ordered":true,"items":["Erstens","Zweitens"]}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b8","type":"divider"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b9","type":"image","caption":"Umsatzkurve"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b10","type":"code","language":"typescript","code":"const x = 1"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b11","type":"simpletable","hasHeader":true,"cells":[["A","B"],["1","2"]]}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b12","type":"quote","text":"Qualität siegt.","attribution":"Team"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b13","type":"pagebreak"}
  ]}]},
  {"columns":[{"width":1,"blocks":[
    {"id":"b14","type":"heading","level":2,"text":"Seite 2"}
  ]}]}
]`

func sampleDocument() *berichte.Document {
	return &berichte.Document{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		Title:     "Q3 Bericht",
		Module:    "finanzen",
		Status:    "draft",
		Rows:      []byte(sampleDocumentRows),
		Settings:  []byte(`{}`),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestDocumentPDFExporter_rendersAllBlockTypes(t *testing.T) {
	t.Parallel()

	results := map[string]*berichte.ReportResult{
		"11111111-1111-1111-1111-111111111111": sampleResult(),
	}

	e := &export.DocumentPDFExporter{}
	var buf bytes.Buffer
	if err := e.Export(sampleDocument(), results, &buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	b := buf.Bytes()
	if len(b) < 4 || string(b[:4]) != "%PDF" {
		t.Fatalf("expected %%PDF prefix, got %d bytes", len(b))
	}
}

func TestDocumentPDFExporter_emptyDocument(t *testing.T) {
	t.Parallel()

	doc := sampleDocument()
	doc.Rows = []byte(`[]`)

	e := &export.DocumentPDFExporter{}
	var buf bytes.Buffer
	if err := e.Export(doc, nil, &buf); err != nil {
		t.Fatalf("Export empty document: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF for an empty document")
	}
}

func TestDocumentPDFExporter_unresolvedChartDoesNotFail(t *testing.T) {
	t.Parallel()

	doc := sampleDocument()
	doc.Rows = []byte(`[{"columns":[{"width":1,"blocks":[
		{"id":"b1","type":"chart","definitionId":"22222222-2222-2222-2222-222222222222","caption":"Fehlt"}
	]}]}]`)

	e := &export.DocumentPDFExporter{}
	var buf bytes.Buffer
	// No entry for the definitionId in the results map — must render the lean
	// placeholder rather than erroring the whole document out.
	if err := e.Export(doc, map[string]*berichte.ReportResult{}, &buf); err != nil {
		t.Fatalf("Export with unresolved chart: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Fatal("expected %PDF prefix")
	}
}

func TestDocumentPDFExporter_malformedRowsErrors(t *testing.T) {
	t.Parallel()

	doc := sampleDocument()
	doc.Rows = []byte(`not json`)

	e := &export.DocumentPDFExporter{}
	var buf bytes.Buffer
	if err := e.Export(doc, nil, &buf); err == nil {
		t.Fatal("expected an error for malformed rows JSON")
	}
}

func TestChartTableDefinitionIDs(t *testing.T) {
	t.Parallel()

	ids, err := export.ChartTableDefinitionIDs([]byte(sampleDocumentRows))
	if err != nil {
		t.Fatalf("ChartTableDefinitionIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected exactly one definition id, got %v", ids)
	}
}

func TestChartTableDefinitionIDs_empty(t *testing.T) {
	t.Parallel()

	ids, err := export.ChartTableDefinitionIDs(nil)
	if err != nil {
		t.Fatalf("ChartTableDefinitionIDs(nil): %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected no ids, got %v", ids)
	}
}
