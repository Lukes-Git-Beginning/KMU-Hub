package testutil

import (
	"context"
	"sort"
	"strings"
	"testing"
)

// systemGlobalAllowlist mirrors the "System-Global Tables (No RLS)" section of
// docs/ARCHITECTURE.md (ADR-006). A table belongs here only once a human has
// written its justification down there — adding a name below without the
// matching doc entry defeats the point of this guard.
var systemGlobalAllowlist = map[string]bool{
	"schema_migrations":    true,
	"caldav_settings":      true,
	"industry_templates":   true,
	"permissions":          true,
	"automation_templates": true,
	"event_types":          true,
	"public_holidays":      true,
}

// knownRLSGaps are tables missing RLS today that are tracked as their own
// backlog unit rather than closed by this guard. Remove an entry the moment
// its unit ships — this map exists so a genuinely NEW gap fails loudly, not
// so an old one goes unnoticed forever.
var knownRLSGaps = map[string]string{
	"user_roles": "backlog unit g-user-roles-rls (docs/ARCHITECTURE.md, \"Offene Luecke\"): " +
		"no tenant_id, no policy; every current read path filters by user_id from the JWT, " +
		"so a direct unfiltered SELECT would be cross-tenant",
}

// TestAllPublicTablesHaveRLSOrAreAllowlisted re-runs the scan that found the
// gaps closed across backlog Block B: every ordinary or partitioned table in
// `public` must either have row security enabled or be named above with a
// reason. It fails the moment a new migration adds an unprotected table.
//
// Partitions (relispartition) are excluded: their partitioned PARENT carries
// the policy, and Postgres evaluates it there, not on the child relation
// (verified for automation_executions and dialer_call_events while scoping
// backlog unit g-rls-events-partition).
func TestAllPublicTablesHaveRLSOrAreAllowlisted(t *testing.T) {
	SkipIfNoDB(t)
	pool := PoolFromEnv(t)
	defer pool.Close()

	rows, err := pool.Query(context.Background(), `
		SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relkind IN ('r', 'p')
		  AND NOT c.relispartition
		  AND NOT c.relrowsecurity
		ORDER BY c.relname`)
	if err != nil {
		t.Fatalf("scan pg_class: %v", err)
	}
	defer rows.Close()

	var unprotected []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		if systemGlobalAllowlist[name] {
			continue
		}
		if reason, known := knownRLSGaps[name]; known {
			t.Logf("known RLS gap, tracked separately: %s — %s", name, reason)
			continue
		}
		unprotected = append(unprotected, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate rows: %v", err)
	}

	if len(unprotected) > 0 {
		sort.Strings(unprotected)
		t.Fatalf(
			"table(s) without RLS and not accounted for: %s\n"+
				"Either run CALL enable_tenant_rls('<table>') / enable_tenant_rls_via_join(...) in a migration, "+
				"or add the table to docs/ARCHITECTURE.md's \"System-Global Tables (No RLS)\" section AND to "+
				"systemGlobalAllowlist in this file with a justification.",
			strings.Join(unprotected, ", "),
		)
	}
}
