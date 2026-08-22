// Package csvutil holds small helpers shared by CSV export code across services.
package csvutil

// NeutralizeFormulaCell defuses CSV formula injection: if val starts with a character
// that Excel/LibreOffice would interpret as the start of a formula (=, +, -, @), a
// leading apostrophe forces the cell to be read as text instead of executed. The value
// itself is never truncated or dropped — export/disclosure obligations still apply.
func NeutralizeFormulaCell(val string) string {
	if val == "" {
		return val
	}
	switch val[0] {
	case '=', '+', '-', '@':
		return "'" + val
	default:
		return val
	}
}
