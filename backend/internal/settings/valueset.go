package settings

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Value-sets are the named option lists the customization editor edits: deal
// pipeline stages, ticket priorities, project statuses. They live in two
// places on purpose:
//
//   - the SYSTEM registry below is the baseline Cosmi ships with. It is code,
//     not data: a release adds or renames a system set, and every tenant sees
//     it without a backfill.
//   - customization_value_sets / _options (migration 000295) hold the TENANT
//     OVERRIDE, and only that. A tenant that never touched a list has no row.
//
// Reading merges the two per option key (see ResolveValueSet), which is what
// the FE mock does across its layers as well
// (desktop/src/renderer/src/mocks/data/customization.ts, resolveValueSet).

// Provenance values mirror ConfigProvenance in
// desktop/src/renderer/src/api/customization-types.ts. "vendor" and "draft"
// exist in that type but are never produced server-side: the vendor layer is
// not writable until R-5 (GDAP vendor access) and drafts live in the editor
// sandbox.
const (
	ProvenanceDefault = "default"
	ProvenanceTenant  = "tenant"
)

// Limits for tenant-supplied value-set input. Generous enough for real lists,
// tight enough that a malformed client cannot write an unbounded document.
const (
	maxValueSetNameLen   = 120
	maxValueSetOptionLen = 120
	maxValueSetColorLen  = 64
	maxValueSetOptions   = 200
)

var (
	// Mirrors the CHECK constraints in migration 000295.
	valueSetKeyPattern    = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)
	valueSetOptionPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
)

// ValueSetOption is one option within a named list. Key is stable and is what
// live records reference (a deal sits in stage "qualified"); renaming edits
// Label. Active=false is a soft delete: hidden from pickers, existing records
// stay intact.
type ValueSetOption struct {
	Key    string
	Label  string
	Color  string // empty = no color token
	Order  int32
	Active bool
}

// ValueSet is one named list, either a system baseline or a tenant override.
type ValueSet struct {
	Key     string
	Name    string
	Options []ValueSetOption
}

// ResolvedValueSetOption carries the layer an effective option came from.
type ResolvedValueSetOption struct {
	ValueSetOption
	Provenance string
}

// ResolvedValueSet is what a read returns: system baseline overlaid with the
// tenant override, options sorted by Order.
type ResolvedValueSet struct {
	Key     string
	Name    string
	Options []ResolvedValueSetOption
	// Provenance of the set definition (its name), not of its options.
	Provenance string
	// IsSystem is derived from the registry, never stored: a shipped list stays
	// a shipped list no matter what the tenant overrode.
	IsSystem bool
}

// systemValueSets is the server-side mirror of DEFAULT_VALUE_SETS in
// desktop/src/renderer/src/mocks/data/customization.ts. Keep the two in sync by
// hand — both are short and change rarely, and a shared source would mean
// generating Go from TypeScript for four literals.
var systemValueSets = map[string]ValueSet{
	"deal_stages": {
		Key:  "deal_stages",
		Name: "Deal-Phasen",
		Options: []ValueSetOption{
			{Key: "lead", Label: "Lead", Color: "hsl(215 16% 47%)", Order: 0, Active: true},
			{Key: "qualified", Label: "Qualifiziert", Color: "hsl(217 91% 60%)", Order: 1, Active: true},
			{Key: "proposal", Label: "Angebot", Color: "hsl(38 92% 50%)", Order: 2, Active: true},
			{Key: "negotiation", Label: "Verhandlung", Color: "hsl(25 95% 53%)", Order: 3, Active: true},
			{Key: "won", Label: "Gewonnen", Color: "hsl(142 71% 45%)", Order: 4, Active: true},
			{Key: "lost", Label: "Verloren", Color: "hsl(0 72% 51%)", Order: 5, Active: true},
		},
	},
	"ticket_priority": {
		Key:  "ticket_priority",
		Name: "Ticket-Priorität",
		Options: []ValueSetOption{
			{Key: "low", Label: "Niedrig", Color: "hsl(142 71% 45%)", Order: 0, Active: true},
			{Key: "medium", Label: "Mittel", Color: "hsl(38 92% 50%)", Order: 1, Active: true},
			{Key: "high", Label: "Hoch", Color: "hsl(25 95% 53%)", Order: 2, Active: true},
			{Key: "critical", Label: "Kritisch", Color: "hsl(0 72% 51%)", Order: 3, Active: true},
		},
	},
	"ticket_status": {
		Key:  "ticket_status",
		Name: "Ticket-Status",
		Options: []ValueSetOption{
			{Key: "open", Label: "Offen", Color: "hsl(38 92% 50%)", Order: 0, Active: true},
			{Key: "in_progress", Label: "In Bearbeitung", Color: "hsl(217 91% 60%)", Order: 1, Active: true},
			{Key: "waiting", Label: "Wartend", Color: "hsl(215 16% 47%)", Order: 2, Active: true},
			{Key: "resolved", Label: "Gelöst", Color: "hsl(142 71% 45%)", Order: 3, Active: true},
			{Key: "closed", Label: "Geschlossen", Color: "hsl(215 16% 47%)", Order: 4, Active: true},
		},
	},
	"project_status": {
		Key:  "project_status",
		Name: "Projekt-Status",
		Options: []ValueSetOption{
			{Key: "planning", Label: "Planung", Color: "hsl(215 16% 47%)", Order: 0, Active: true},
			{Key: "active", Label: "Aktiv", Color: "hsl(217 91% 60%)", Order: 1, Active: true},
			{Key: "on_hold", Label: "Pausiert", Color: "hsl(38 92% 50%)", Order: 2, Active: true},
			{Key: "completed", Label: "Abgeschlossen", Color: "hsl(142 71% 45%)", Order: 3, Active: true},
			{Key: "cancelled", Label: "Abgebrochen", Color: "hsl(0 72% 51%)", Order: 4, Active: true},
		},
	},
}

// IsSystemValueSet reports whether the key names a list Cosmi ships with.
// Such a list cannot be deleted — deleting only drops the tenant's override
// and falls the list back to its shipped baseline.
func IsSystemValueSet(key string) bool {
	_, ok := systemValueSets[key]
	return ok
}

// SystemValueSet returns a copy of the shipped baseline for key.
func SystemValueSet(key string) (ValueSet, bool) {
	set, ok := systemValueSets[key]
	if !ok {
		return ValueSet{}, false
	}
	return cloneValueSet(set), true
}

// SystemValueSetKeys returns the shipped list keys, sorted.
func SystemValueSetKeys() []string {
	keys := make([]string, 0, len(systemValueSets))
	for k := range systemValueSets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func cloneValueSet(set ValueSet) ValueSet {
	out := ValueSet{Key: set.Key, Name: set.Name}
	out.Options = append(out.Options, set.Options...)
	return out
}

// ResolveValueSet overlays a tenant override onto the shipped baseline.
//
// The merge is per option KEY, not wholesale: an option the tenant did not
// mention keeps its shipped definition and provenance "default". That is why a
// tenant hides a shipped option by sending it with Active=false rather than by
// omitting it — omission cannot mean "delete" for a key that live records may
// still point at.
//
// Returns nil when neither layer knows the key.
func ResolveValueSet(key string, override *ValueSet) *ResolvedValueSet {
	base, hasBase := systemValueSets[key]
	if !hasBase && override == nil {
		return nil
	}

	merged := make(map[string]ResolvedValueSetOption, len(base.Options))
	order := make([]string, 0, len(base.Options))

	if hasBase {
		for _, opt := range base.Options {
			merged[opt.Key] = ResolvedValueSetOption{ValueSetOption: opt, Provenance: ProvenanceDefault}
			order = append(order, opt.Key)
		}
	}
	if override != nil {
		for _, opt := range override.Options {
			if _, seen := merged[opt.Key]; !seen {
				order = append(order, opt.Key)
			}
			merged[opt.Key] = ResolvedValueSetOption{ValueSetOption: opt, Provenance: ProvenanceTenant}
		}
	}

	resolved := &ResolvedValueSet{
		Key:        key,
		Name:       base.Name,
		Provenance: ProvenanceDefault,
		IsSystem:   hasBase,
		Options:    make([]ResolvedValueSetOption, 0, len(merged)),
	}
	if override != nil && override.Name != "" {
		resolved.Name = override.Name
		resolved.Provenance = ProvenanceTenant
	}
	if resolved.Name == "" {
		resolved.Name = key
	}

	for _, k := range order {
		resolved.Options = append(resolved.Options, merged[k])
	}
	// Stable sort on Order; `order` keeps equal-Order options in baseline-then-
	// tenant sequence instead of Go's map iteration randomness.
	sort.SliceStable(resolved.Options, func(i, j int) bool {
		return resolved.Options[i].Order < resolved.Options[j].Order
	})
	return resolved
}

// BaseValueSet returns the shipped baseline as a resolved set (the editor's
// "?base=1" view), or nil for a key that has no baseline.
func BaseValueSet(key string) *ResolvedValueSet {
	base, ok := systemValueSets[key]
	if !ok {
		return nil
	}
	resolved := &ResolvedValueSet{
		Key:        key,
		Name:       base.Name,
		Provenance: ProvenanceDefault,
		IsSystem:   true,
		Options:    make([]ResolvedValueSetOption, 0, len(base.Options)),
	}
	for _, opt := range base.Options {
		resolved.Options = append(resolved.Options, ResolvedValueSetOption{ValueSetOption: opt, Provenance: ProvenanceDefault})
	}
	return resolved
}

// validateValueSet checks tenant-supplied input at the service boundary. The
// DB CHECKs say the same things; failing here turns a 23514 into a precise
// InvalidArgument the editor can show next to the offending field.
func validateValueSet(set *ValueSet) error {
	if set == nil {
		return ErrInvalidValueSet
	}
	if !valueSetKeyPattern.MatchString(set.Key) {
		return fmt.Errorf("%w: key %q", ErrInvalidValueSetKey, set.Key)
	}
	name := strings.TrimSpace(set.Name)
	if name == "" || len([]rune(name)) > maxValueSetNameLen {
		return fmt.Errorf("%w: name must be 1..%d characters", ErrInvalidValueSet, maxValueSetNameLen)
	}
	if len(set.Options) == 0 {
		return fmt.Errorf("%w: at least one option is required", ErrInvalidValueSet)
	}
	if len(set.Options) > maxValueSetOptions {
		return fmt.Errorf("%w: at most %d options", ErrInvalidValueSet, maxValueSetOptions)
	}

	seen := make(map[string]bool, len(set.Options))
	for _, opt := range set.Options {
		if !valueSetOptionPattern.MatchString(opt.Key) {
			return fmt.Errorf("%w: option key %q", ErrInvalidValueSetOption, opt.Key)
		}
		if seen[opt.Key] {
			return fmt.Errorf("%w: duplicate option key %q", ErrInvalidValueSetOption, opt.Key)
		}
		seen[opt.Key] = true

		label := strings.TrimSpace(opt.Label)
		if label == "" || len([]rune(label)) > maxValueSetOptionLen {
			return fmt.Errorf("%w: label of %q must be 1..%d characters", ErrInvalidValueSetOption, opt.Key, maxValueSetOptionLen)
		}
		if len([]rune(opt.Color)) > maxValueSetColorLen {
			return fmt.Errorf("%w: color of %q exceeds %d characters", ErrInvalidValueSetOption, opt.Key, maxValueSetColorLen)
		}
	}
	return nil
}
