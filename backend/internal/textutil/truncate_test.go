package textutil

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateBytes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"shorter than limit", "abc", 5, "abc"},
		{"exactly at limit", "abcde", 5, "abcde"},
		{"ascii cut", "abcdefg", 5, "abcde"},
		{"cut lands between the two bytes of a umlaut", "abcd" + "ä" + "x", 5, "abcd"},
		{"cut lands right after a umlaut", "abc" + "ä" + "x", 5, "abcä"},
		{"three byte rune is dropped whole", "ab" + "€" + "x", 4, "ab"},
		{"zero limit", "abc", 0, ""},
		{"negative limit", "abc", -1, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TruncateBytes(tc.in, tc.max)
			if got != tc.want {
				t.Errorf("TruncateBytes(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("result %q is not valid UTF-8", got)
			}
		})
	}
}

// The 200-byte preview limit used by the mail sync and the inbox adapters:
// a subject whose byte 200 falls inside a umlaut must stay valid.
func TestTruncateBytes_PreviewBoundaryInsideUmlaut(t *testing.T) {
	subject := strings.Repeat("a", 199) + "ü" + "rest"
	got := TruncateBytes(subject, 200)
	if !utf8.ValidString(got) {
		t.Fatalf("preview is not valid UTF-8: %q", got)
	}
	if len(got) > 200 {
		t.Fatalf("preview has %d bytes, want <= 200", len(got))
	}
	if got != strings.Repeat("a", 199) {
		t.Fatalf("unexpected preview %q", got)
	}
}
