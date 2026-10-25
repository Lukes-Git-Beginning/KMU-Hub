package gateway

// Wire shape of the audit, vault and password policy routes as the security pages type them
// (security-types.ts: AuditEntry, VaultSecretValue, PasswordPolicy, PasswordValidationResult).
// The gRPC service keeps its proto types; the conversion sits at this edge only.

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	securityv1 "github.com/kmuhub/kmuhub/proto/security/v1"
)

// auditPageSizeMax caps one audit page: the log page asks for 25 rows, the roles tab for 200. The
// audit service caps at 1000, so the gateway ceiling is the tighter one.
const auditPageSizeMax = 200

type auditEntryWire struct {
	ID           string          `json:"id"`
	SequenceNum  string          `json:"sequence_num"`
	Timestamp    time.Time       `json:"timestamp"`
	UserID       string          `json:"user_id"`
	Action       string          `json:"action"`
	Target       string          `json:"target"`
	TargetType   string          `json:"target_type"`
	Details      json.RawMessage `json:"details"`
	IPAddress    string          `json:"ip_address"`
	UserAgent    string          `json:"user_agent"`
	Result       string          `json:"result"`
	PreviousHash string          `json:"previous_hash"`
	EntryHash    string          `json:"entry_hash"`
}

// auditDetailsObject returns the JSON object stored in AuditEntry.details. The detail dialog
// tests `'old_value' in entry.details`, which throws for a string, so anything that is not a
// JSON object (empty, malformed, a bare array) becomes `{}`.
func auditDetailsObject(s string) json.RawMessage {
	var obj map[string]json.RawMessage
	if json.Unmarshal([]byte(s), &obj) != nil || obj == nil {
		return json.RawMessage("{}")
	}
	return json.RawMessage(s)
}

type auditListWire struct {
	Entries []auditEntryWire `json:"entries"`
	Total   int32            `json:"total"`
}

func toAuditListWire(resp *securityv1.ListAuditEntriesResponse) auditListWire {
	entries := make([]auditEntryWire, 0, len(resp.GetEntries()))
	for _, e := range resp.GetEntries() {
		entries = append(entries, auditEntryWire{
			ID:           e.GetId(),
			SequenceNum:  strconv.FormatInt(e.GetSequenceNum(), 10),
			Timestamp:    e.GetTimestamp().AsTime(),
			UserID:       e.GetUserId(),
			Action:       e.GetAction(),
			Target:       e.GetTarget(),
			TargetType:   e.GetTargetType(),
			Details:      auditDetailsObject(e.GetDetails()),
			IPAddress:    e.GetIpAddress(),
			UserAgent:    e.GetUserAgent(),
			Result:       e.GetResult(),
			PreviousHash: e.GetPreviousHash(),
			EntryHash:    e.GetEntryHash(),
		})
	}
	return auditListWire{Entries: entries, Total: resp.GetTotal()}
}

// auditFilterFromQuery reads the filter the log page and the export send: `date_from`/`date_to`
// (YYYY-MM-DD or RFC3339), `user_id`, `action`, `result`. A bare `date_to` day is inclusive:
// the repository compares `timestamp <= date_to`, so midnight would drop that whole day.
// A malformed date writes a 400 and returns ok=false.
func auditFilterFromQuery(w http.ResponseWriter, r *http.Request) (*securityv1.AuditFilter, bool) {
	q := r.URL.Query()
	filter := &securityv1.AuditFilter{
		Action: q.Get("action"),
		Result: q.Get("result"),
		UserId: q.Get("user_id"),
	}
	from, ok := parseDateField(w, "date_from", q.Get("date_from"))
	if !ok {
		return nil, false
	}
	to, ok := parseDateField(w, "date_to", q.Get("date_to"))
	if !ok {
		return nil, false
	}
	if to != nil && len(q.Get("date_to")) == len("2006-01-02") {
		to = timestamppb.New(to.AsTime().Add(24*time.Hour - time.Nanosecond))
	}
	filter.DateFrom, filter.DateTo = from, to
	return filter, true
}

// auditPageFromQuery reads the window the log page sends (`offset`/`limit`) and falls back to
// `page`/`page_size` for callers that page by number.
func auditPageFromQuery(r *http.Request) (offset, limit int32) {
	q := r.URL.Query()
	page, pageSize := parsePagination(r, 1, 50)
	offset, limit = int32((page-1)*pageSize), int32(pageSize)
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 {
		v = min(v, auditPageSizeMax) // clamp, not fall back to the default: limit=500 asks for "as many as allowed"
		limit = int32(v)
		if q.Get("offset") == "" {
			offset = int32((page - 1) * v)
		}
	}
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v >= 0 {
		offset = int32(v)
	}
	return offset, limit
}

type vaultSecretValueWire struct {
	KeyName string `json:"key_name"`
	Value   string `json:"value"`
}

type passwordPolicyWire struct {
	ID                string     `json:"id"`
	MinLength         int32      `json:"min_length"`
	MinEntropy        float64    `json:"min_entropy"`
	MaxAgeDays        *int32     `json:"max_age_days"`
	PreventReuseCount int32      `json:"prevent_reuse_count"`
	RequireUppercase  bool       `json:"require_uppercase"`
	RequireLowercase  bool       `json:"require_lowercase"`
	RequireDigit      bool       `json:"require_digit"`
	RequireSpecial    bool       `json:"require_special"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
	UpdatedBy         string     `json:"updated_by,omitempty"`
}

// toPasswordPolicyWire maps `max_age_days` 0 ("no expiration") to null, the type's
// `number | null`.
func toPasswordPolicyWire(p *securityv1.PasswordPolicy) passwordPolicyWire {
	wire := passwordPolicyWire{
		ID:                p.GetId(),
		MinLength:         p.GetMinLength(),
		MinEntropy:        p.GetMinEntropy(),
		PreventReuseCount: p.GetPreventReuseCount(),
		RequireUppercase:  p.GetRequireUppercase(),
		RequireLowercase:  p.GetRequireLowercase(),
		RequireDigit:      p.GetRequireDigit(),
		RequireSpecial:    p.GetRequireSpecial(),
		UpdatedBy:         p.GetUpdatedBy(),
	}
	if days := p.GetMaxAgeDays(); days > 0 {
		wire.MaxAgeDays = &days
	}
	if p.GetUpdatedAt() != nil {
		t := p.GetUpdatedAt().AsTime()
		wire.UpdatedAt = &t
	}
	return wire
}

// passwordPolicyPatch is the PUT body. A field that is absent keeps its current value; a
// `max_age_days` of null means "no expiration" (the form sends null for 0).
type passwordPolicyPatch struct {
	MinLength         *int32          `json:"min_length" validate:"omitempty,gte=1,lte=128"`
	RequireUppercase  *bool           `json:"require_uppercase"`
	RequireLowercase  *bool           `json:"require_lowercase"`
	RequireDigit      *bool           `json:"require_digit"`
	RequireSpecial    *bool           `json:"require_special"`
	MinEntropy        *float64        `json:"min_entropy"`
	MaxAgeDays        json.RawMessage `json:"max_age_days"`
	PreventReuseCount *int32          `json:"prevent_reuse_count"`
}

// complete reports whether every field was sent, i.e. whether the current policy is not needed.
func (p *passwordPolicyPatch) complete() bool {
	return p.MinLength != nil && p.RequireUppercase != nil && p.RequireLowercase != nil &&
		p.RequireDigit != nil && p.RequireSpecial != nil && p.MinEntropy != nil &&
		len(p.MaxAgeDays) > 0 && p.PreventReuseCount != nil
}

// applyTo overlays the sent fields on base and returns the policy to store. ok=false means
// `max_age_days` was neither a number nor null.
func (p *passwordPolicyPatch) applyTo(base *securityv1.PasswordPolicy) (*securityv1.PasswordPolicy, bool) {
	out := &securityv1.PasswordPolicy{
		MinLength:         base.GetMinLength(),
		RequireUppercase:  base.GetRequireUppercase(),
		RequireLowercase:  base.GetRequireLowercase(),
		RequireDigit:      base.GetRequireDigit(),
		RequireSpecial:    base.GetRequireSpecial(),
		MinEntropy:        base.GetMinEntropy(),
		MaxAgeDays:        base.GetMaxAgeDays(),
		PreventReuseCount: base.GetPreventReuseCount(),
	}
	if p.MinLength != nil {
		out.MinLength = *p.MinLength
	}
	if p.RequireUppercase != nil {
		out.RequireUppercase = *p.RequireUppercase
	}
	if p.RequireLowercase != nil {
		out.RequireLowercase = *p.RequireLowercase
	}
	if p.RequireDigit != nil {
		out.RequireDigit = *p.RequireDigit
	}
	if p.RequireSpecial != nil {
		out.RequireSpecial = *p.RequireSpecial
	}
	if p.MinEntropy != nil {
		out.MinEntropy = *p.MinEntropy
	}
	if p.PreventReuseCount != nil {
		out.PreventReuseCount = *p.PreventReuseCount
	}
	if len(p.MaxAgeDays) > 0 {
		var days *int32
		if json.Unmarshal(p.MaxAgeDays, &days) != nil {
			return nil, false
		}
		out.MaxAgeDays = 0
		if days != nil {
			out.MaxAgeDays = *days
		}
	}
	return out, true
}
