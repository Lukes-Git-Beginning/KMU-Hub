package wiki

import (
	"strings"
	"unicode"
)

// BuildSearchQuery converts a user-supplied search string into a PostgreSQL
// tsquery expression.
//
// Rules:
//   - Each whitespace-separated token becomes an AND operand.
//   - The last token gets a prefix-wildcard (:*) so partial words match.
//   - Tokens that would produce an empty string after cleaning are skipped.
//
// Example: "Kunden report" -> "Kunden & report:*"
//
// Note: the returned string is always used as a parameterised query argument,
// so there is no SQL-injection risk. Cleaning is done solely to produce a
// syntactically valid tsquery.
func BuildSearchQuery(query string) string {
	words := strings.FieldsFunc(query, func(r rune) bool {
		return unicode.IsSpace(r)
	})

	var tokens []string
	for _, w := range words {
		cleaned := cleanToken(w)
		if cleaned != "" {
			tokens = append(tokens, cleaned)
		}
	}

	if len(tokens) == 0 {
		return ""
	}

	// Add prefix-wildcard to the last token
	tokens[len(tokens)-1] = tokens[len(tokens)-1] + ":*"

	return strings.Join(tokens, " & ")
}

// cleanToken strips characters that are illegal in a tsquery lexeme.
// Keeps letters, digits, hyphens and underscores.
// Leading/trailing hyphens are removed to avoid invalid tsquery syntax.
func cleanToken(token string) string {
	var b strings.Builder
	for _, r := range token {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}
