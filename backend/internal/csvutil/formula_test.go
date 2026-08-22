package csvutil

import "testing"

func TestNeutralizeFormulaCell(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"=HYPERLINK(\"http://evil.example/\",\"x\")", "'=HYPERLINK(\"http://evil.example/\",\"x\")"},
		{"+cmd|'/C calc'!A0", "'+cmd|'/C calc'!A0"},
		{"-2+3", "'-2+3"},
		{"@SUM(1,2)", "'@SUM(1,2)"},
		{"Normal value", "Normal value"},
		{"user@example.com", "user@example.com"},
	}
	for _, c := range cases {
		if got := NeutralizeFormulaCell(c.in); got != c.want {
			t.Errorf("NeutralizeFormulaCell(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
