package export

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/page"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// DocumentPDFExporter renders a berichte.Document's block tree
// (rows -> columns -> blocks, see models.Document.Rows) as an A4 PDF using
// maroto/v2 — the same renderer family as PDFExporter, no new dependency.
//
// rows/settings are opaque JSONB (see the JSONB-passthrough decision in
// BACKLOG.yml p3-berichte-document-persistence): the editor owns the block
// shapes, and this file mirrors them as unexported Go structs with the same
// camelCase JSON tags rather than sharing types with the proto layer.
type DocumentPDFExporter struct{}

// Export writes the PDF representation of doc into w. chartResults must carry
// one resolved *berichte.ReportResult per distinct definitionId returned by
// ChartTableDefinitionIDs — the caller (BerichteGRPCServer) resolves those via
// Service.RunReport before calling Export, keeping this package free of a
// dependency on the service layer.
func (e *DocumentPDFExporter) Export(doc *berichte.Document, chartResults map[string]*berichte.ReportResult, w io.Writer) error {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(10).
		WithTopMargin(10).
		WithRightMargin(10).
		Build()

	var docRows []documentRow
	if len(doc.Rows) > 0 {
		if err := json.Unmarshal(doc.Rows, &docRows); err != nil {
			return fmt.Errorf("parse document rows: %w", err)
		}
	}

	var pages [][]core.Row
	var current []core.Row
	flush := func() {
		pages = append(pages, current)
		current = nil
	}

	for _, dr := range docRows {
		if columnsAreKPIRow(dr.Columns) {
			kpiRows, err := renderKPIRow(dr.Columns)
			if err != nil {
				return err
			}
			current = append(current, kpiRows...)
			continue
		}

		// lean: side-by-side columns are only laid out for the documented
		// "KPI row" case above. A row that mixes column counts/content (e.g.
		// "text beside chart") renders every block full-width in column
		// order instead of a real multi-column layout — maroto's row model
		// shares one height across all cols in a row, and a general column
		// engine (independent per-column flow, balanced heights) is real
		// layout work. Upgrade when customers report needing side-by-side
		// text/chart in the exported PDF.
		for _, col := range dr.Columns {
			for _, raw := range col.Blocks {
				var base documentBlockBase
				if err := json.Unmarshal(raw, &base); err != nil {
					return fmt.Errorf("parse document block: %w", err)
				}
				if base.Type == "pagebreak" {
					flush()
					continue
				}
				blockRows, err := renderDocumentBlock(base.Type, raw, chartResults)
				if err != nil {
					return err
				}
				current = append(current, blockRows...)
			}
		}
	}
	flush()

	m := maroto.New(cfg)
	for _, pr := range pages {
		if len(pr) == 0 {
			continue
		}
		m.AddPages(page.New().Add(pr...))
	}

	generated, err := m.Generate()
	if err != nil {
		return fmt.Errorf("pdf generate: %w", err)
	}
	_, err = w.Write(generated.GetBytes())
	return err
}

// ============================================================================
// Block tree — mirrors berichte-types.ts (ReportRow/ReportDocColumn/ReportBlock)
// ============================================================================

type documentRow struct {
	Columns []documentColumn `json:"columns"`
}

type documentColumn struct {
	Width  float64           `json:"width"`
	Blocks []json.RawMessage `json:"blocks"`
}

type documentBlockBase struct {
	Type string `json:"type"`
}

type coverBlockData struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Author   string `json:"author"`
	ShowDate bool   `json:"showDate"`
}

type headingBlockData struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

type textBlockData struct {
	HTML string `json:"html"`
}

type chartTableBlockData struct {
	DefinitionID string `json:"definitionId"`
	Caption      string `json:"caption"`
}

type kpiBlockData struct {
	Label         string   `json:"label"`
	Value         string   `json:"value"`
	Unit          string   `json:"unit"`
	ChangePercent *float64 `json:"changePercent"`
}

type calloutBlockData struct {
	Variant string `json:"variant"`
	Title   string `json:"title"`
	HTML    string `json:"html"`
}

type bulletBlockData struct {
	Items   []string `json:"items"`
	Ordered bool     `json:"ordered"`
}

type imageBlockData struct {
	URL     string `json:"url"`
	Caption string `json:"caption"`
	Alt     string `json:"alt"`
}

type codeBlockData struct {
	Code string `json:"code"`
}

type simpleTableBlockData struct {
	Cells     [][]string `json:"cells"`
	HasHeader *bool      `json:"hasHeader"`
}

type quoteBlockData struct {
	Text        string `json:"text"`
	Attribution string `json:"attribution"`
}

// ChartTableDefinitionIDs returns the distinct saved-definition ids referenced
// by chart/table blocks in the document's row tree (blocks with an inline
// query instead of a definitionId are not returned — see the lean note in
// renderChartTableBlock). The caller resolves each id via
// berichte.Service.RunReport before calling Export.
func ChartTableDefinitionIDs(rowsJSON []byte) ([]string, error) {
	if len(rowsJSON) == 0 {
		return nil, nil
	}
	var docRows []documentRow
	if err := json.Unmarshal(rowsJSON, &docRows); err != nil {
		return nil, fmt.Errorf("parse document rows: %w", err)
	}

	seen := make(map[string]bool)
	var ids []string
	for _, dr := range docRows {
		for _, col := range dr.Columns {
			for _, raw := range col.Blocks {
				var base documentBlockBase
				if err := json.Unmarshal(raw, &base); err != nil {
					continue
				}
				if base.Type != "chart" && base.Type != "table" {
					continue
				}
				var b chartTableBlockData
				if err := json.Unmarshal(raw, &b); err != nil || b.DefinitionID == "" {
					continue
				}
				if seen[b.DefinitionID] {
					continue
				}
				seen[b.DefinitionID] = true
				ids = append(ids, b.DefinitionID)
			}
		}
	}
	return ids, nil
}

// ============================================================================
// Rendering
// ============================================================================

var (
	docGrey      = &props.Color{Red: 120, Green: 120, Blue: 120}
	docLightGrey = &props.Color{Red: 245, Green: 245, Blue: 245}
	docCodeBg    = &props.Color{Red: 240, Green: 240, Blue: 240}
	docImageBg   = &props.Color{Red: 250, Green: 250, Blue: 250}
)

func columnsAreKPIRow(columns []documentColumn) bool {
	if len(columns) < 2 {
		return false
	}
	for _, c := range columns {
		if len(c.Blocks) != 1 {
			return false
		}
		var base documentBlockBase
		if err := json.Unmarshal(c.Blocks[0], &base); err != nil || base.Type != "kpi" {
			return false
		}
	}
	return true
}

// gridWidths distributes column weights (default 1) across the 12-grid,
// widening the last column so the sizes always sum to exactly 12.
func gridWidths(columns []documentColumn) []int {
	n := len(columns)
	widths := make([]int, n)
	sum := 0.0
	weights := make([]float64, n)
	for i, c := range columns {
		w := c.Width
		if w <= 0 {
			w = 1
		}
		weights[i] = w
		sum += w
	}
	used := 0
	for i, w := range weights {
		widths[i] = int(w / sum * 12)
		if widths[i] < 1 {
			widths[i] = 1
		}
		used += widths[i]
	}
	widths[n-1] += 12 - used
	return widths
}

func renderKPIRow(columns []documentColumn) ([]core.Row, error) {
	kpis := make([]kpiBlockData, len(columns))
	for i, c := range columns {
		if err := json.Unmarshal(c.Blocks[0], &kpis[i]); err != nil {
			return nil, fmt.Errorf("parse kpi block: %w", err)
		}
	}
	widths := gridWidths(columns)
	return renderKPICards(kpis, widths), nil
}

// renderKPICards lays out one metric per column across three stacked rows
// (label/value/change) sharing a light-grey background — a col in maroto
// renders every added component at the same position, so each metric line
// must be its own row rather than stacked components within one col.
func renderKPICards(kpis []kpiBlockData, widths []int) []core.Row {
	labelCols := make([]core.Col, len(kpis))
	valueCols := make([]core.Col, len(kpis))
	changeCols := make([]core.Col, len(kpis))
	for i, k := range kpis {
		labelCols[i] = col.New(widths[i]).Add(text.New(k.Label, props.Text{Size: 8, Color: docGrey}))

		valueText := k.Value
		if k.Unit != "" {
			valueText += " " + k.Unit
		}
		valueCols[i] = col.New(widths[i]).Add(text.New(valueText, props.Text{Size: 14, Style: fontstyle.Bold}))

		changeText := ""
		if k.ChangePercent != nil {
			changeText = formatChangePercent(*k.ChangePercent)
		}
		changeCols[i] = col.New(widths[i]).Add(text.New(changeText, props.Text{Size: 8, Color: changeColor(k.ChangePercent)}))
	}
	bg := &props.Cell{BackgroundColor: docLightGrey}
	return []core.Row{
		row.New(6).WithStyle(bg).Add(labelCols...),
		row.New(9).WithStyle(bg).Add(valueCols...),
		row.New(6).WithStyle(bg).Add(changeCols...),
	}
}

func formatChangePercent(pct float64) string {
	sign := "+"
	if pct < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.1f %%", sign, pct)
}

func changeColor(pct *float64) *props.Color {
	if pct == nil {
		return docGrey
	}
	if *pct < 0 {
		return &props.Color{Red: 180, Green: 40, Blue: 40}
	}
	return &props.Color{Red: 30, Green: 130, Blue: 70}
}

func calloutColor(variant string) *props.Color {
	switch variant {
	case "success":
		return &props.Color{Red: 224, Green: 242, Blue: 229}
	case "warning":
		return &props.Color{Red: 255, Green: 243, Blue: 205}
	case "recommendation":
		return &props.Color{Red: 237, Green: 231, Blue: 246}
	default: // "info" and unknown
		return &props.Color{Red: 224, Green: 236, Blue: 250}
	}
}

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// stripHTML reduces TipTap-authored HTML to plain text: tags become spaces,
// entities are unescaped, and runs of whitespace collapse to one space.
//
// lean: this drops all formatting (bold/italic/lists nested inside a
// paragraph) instead of mapping TipTap's HTML into styled maroto text runs —
// maroto's text component has no rich-text renderer, and that mapping is real
// work. Upgrade when customers ask for formatting to survive the PDF export.
func stripHTML(s string) string {
	s = htmlTagPattern.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.Join(strings.Fields(s), " ")
}

func renderDocumentBlock(blockType string, raw json.RawMessage, chartResults map[string]*berichte.ReportResult) ([]core.Row, error) {
	switch blockType {
	case "cover":
		var b coverBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse cover block: %w", err)
		}
		return renderCover(b), nil

	case "heading":
		var b headingBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse heading block: %w", err)
		}
		size := 14.0
		if b.Level <= 1 {
			size = 18
		}
		return []core.Row{
			text.NewAutoRow(b.Text, props.Text{Style: fontstyle.Bold, Size: size, Top: 6, Bottom: 3}),
		}, nil

	case "text":
		var b textBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse text block: %w", err)
		}
		return []core.Row{
			text.NewAutoRow(stripHTML(b.HTML), props.Text{Size: 10, Top: 2, Bottom: 4}),
		}, nil

	case "chart", "table":
		var b chartTableBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse %s block: %w", blockType, err)
		}
		return renderChartTableBlock(blockType, b, chartResults), nil

	case "kpi":
		var b kpiBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse kpi block: %w", err)
		}
		return renderKPICards([]kpiBlockData{b}, []int{12}), nil

	case "callout":
		var b calloutBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse callout block: %w", err)
		}
		return renderCallout(b), nil

	case "bullet":
		var b bulletBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse bullet block: %w", err)
		}
		return renderBullets(b), nil

	case "divider":
		return []core.Row{
			line.NewRow(4, props.Line{Color: &props.Color{Red: 210, Green: 210, Blue: 210}}),
		}, nil

	case "image":
		var b imageBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse image block: %w", err)
		}
		return []core.Row{renderImagePlaceholder(b)}, nil

	case "pagebreak":
		// Handled by the caller (Export), which flushes the current page
		// instead of rendering content for this block.
		return nil, nil

	case "code":
		var b codeBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse code block: %w", err)
		}
		return []core.Row{
			text.NewAutoRow(b.Code, props.Text{Family: fontfamily.Courier, Size: 8, Left: 3, Top: 2, Bottom: 2}).
				WithStyle(&props.Cell{BackgroundColor: docCodeBg}),
		}, nil

	case "simpletable":
		var b simpleTableBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse simpletable block: %w", err)
		}
		return renderSimpleTable(b), nil

	case "quote":
		var b quoteBlockData
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, fmt.Errorf("parse quote block: %w", err)
		}
		return renderQuote(b), nil

	default:
		return []core.Row{
			text.NewAutoRow(fmt.Sprintf("[unbekannter Blocktyp: %s]", blockType), props.Text{Size: 8, Color: docGrey}),
		}, nil
	}
}

func renderCover(b coverBlockData) []core.Row {
	rows := []core.Row{
		text.NewAutoRow(b.Title, props.Text{Style: fontstyle.Bold, Size: 22, Align: align.Center, Top: 20, Bottom: 4}),
	}
	if b.Subtitle != "" {
		rows = append(rows, text.NewAutoRow(b.Subtitle, props.Text{Size: 12, Align: align.Center, Color: docGrey, Bottom: 8}))
	}
	if b.Author != "" {
		rows = append(rows, text.NewAutoRow(b.Author, props.Text{Size: 10, Align: align.Center, Color: docGrey}))
	}
	return rows
}

func renderCallout(b calloutBlockData) []core.Row {
	bg := &props.Cell{BackgroundColor: calloutColor(b.Variant)}
	var rows []core.Row
	if b.Title != "" {
		rows = append(rows, text.NewAutoRow(b.Title, props.Text{Style: fontstyle.Bold, Size: 10, Left: 3, Top: 3}).WithStyle(bg))
	}
	rows = append(rows, text.NewAutoRow(stripHTML(b.HTML), props.Text{Size: 9, Left: 3, Bottom: 3}).WithStyle(bg))
	return rows
}

func renderBullets(b bulletBlockData) []core.Row {
	rows := make([]core.Row, 0, len(b.Items))
	for i, item := range b.Items {
		prefix := "•"
		if b.Ordered {
			prefix = fmt.Sprintf("%d.", i+1)
		}
		rows = append(rows, text.NewAutoRow(prefix+" "+item, props.Text{Size: 9, Left: 4, Bottom: 1}))
	}
	return rows
}

// renderImagePlaceholder names the image instead of embedding it.
//
// lean: fetching an arbitrary block-stored URL at export time is an SSRF
// surface this iteration does not want to open, and embedding a MinIO-backed
// asset needs its own auth/presign path. Upgrade when image embedding in the
// PDF export is worth that review.
func renderImagePlaceholder(b imageBlockData) core.Row {
	label := b.Caption
	if label == "" {
		label = b.Alt
	}
	if label == "" {
		label = b.URL
	}
	if label == "" {
		label = "Bild"
	}
	return text.NewAutoRow("[Bild: "+label+"]", props.Text{Size: 9, Align: align.Center, Color: docGrey, Top: 4, Bottom: 4}).
		WithStyle(&props.Cell{BackgroundColor: docImageBg})
}

func renderQuote(b quoteBlockData) []core.Row {
	rows := []core.Row{
		text.NewAutoRow(b.Text, props.Text{Style: fontstyle.Italic, Size: 10, Left: 6, Top: 3}),
	}
	if b.Attribution != "" {
		rows = append(rows, text.NewAutoRow("— "+b.Attribution, props.Text{Size: 8, Left: 6, Color: docGrey, Bottom: 3}))
	}
	return rows
}

func renderSimpleTable(b simpleTableBlockData) []core.Row {
	if len(b.Cells) == 0 {
		return nil
	}
	hasHeader := b.HasHeader == nil || *b.HasHeader

	numCols := len(b.Cells[0])
	if numCols == 0 {
		numCols = 1
	}
	colWidth := 12 / numCols
	if colWidth < 1 {
		colWidth = 1
	}

	rows := make([]core.Row, 0, len(b.Cells))
	for i, cells := range b.Cells {
		isHeader := hasHeader && i == 0
		cols := make([]core.Col, 0, numCols)
		for j := 0; j < numCols; j++ {
			val := ""
			if j < len(cells) {
				val = cells[j]
			}
			style := props.Text{Size: 8}
			if isHeader {
				style.Style = fontstyle.Bold
				style.Color = &props.Color{Red: 255, Green: 255, Blue: 255}
			}
			cols = append(cols, col.New(colWidth).Add(text.New(val, style)))
		}

		r := row.New(7)
		switch {
		case isHeader:
			r = r.WithStyle(&props.Cell{BackgroundColor: &props.Color{Red: 50, Green: 50, Blue: 50}})
		case i%2 == 0:
			r = r.WithStyle(&props.Cell{BackgroundColor: docLightGrey})
		}
		rows = append(rows, r.Add(cols...))
	}
	return rows
}

// renderChartTableBlock renders the chart/table block's resolved data as a
// table (decision: chart/table blocks render as a data table, not a graphic —
// see the decision note on p3-berichte-server-pdf in BACKLOG.yml. No chart
// visualization dependency (go-chart, gonum/plot) is added; a `lean:` upgrade
// trigger is recorded there, not duplicated here).
func renderChartTableBlock(kind string, b chartTableBlockData, chartResults map[string]*berichte.ReportResult) []core.Row {
	label := "Diagramm"
	if kind == "table" {
		label = "Tabelle"
	}

	var rows []core.Row
	if b.Caption != "" {
		rows = append(rows, text.NewAutoRow(b.Caption, props.Text{Style: fontstyle.Bold, Size: 10, Top: 4, Bottom: 1}))
	}

	result := chartResults[b.DefinitionID]
	if b.DefinitionID == "" || result == nil {
		// lean: blocks with an inline query_config instead of a saved
		// definitionId have no server-side execution path — the executor
		// only runs saved definitions (see backend-gaps.md "Query-Builder:
		// BE-Executor liest query_config schon — Editor-Contract
		// festzurren"). A definitionId that failed to resolve (deleted
		// definition, downstream error) lands here too. Upgrade once inline
		// query execution exists for the block editor.
		rows = append(rows, text.NewAutoRow(
			fmt.Sprintf("[%s ohne gespeicherte Definition — im PDF-Export nicht darstellbar]", label),
			props.Text{Size: 9, Color: docGrey},
		))
		return rows
	}

	rows = append(rows, resultRows(result)...)
	return rows
}
