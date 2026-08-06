package helpdesk

import (
	"net/mail"
	"strings"
)

// TicketIntake carries where a ticket came from and what the intake form
// collected beyond the fixed ticket fields. It is grouped into one value
// rather than five more positional parameters on CreateTicket, which already
// takes eleven -- a sixteen-argument call is an argument-order bug waiting to
// be written.
//
// The module has been sending all of this on every intake create for a while;
// the gateway DTO dropped it silently and answered 200 (BACKLOG B2).
type TicketIntake struct {
	// Channel is the intake origin: "agent" | "selfservice" | "external".
	// Empty means "agent"; anything else is rejected rather than defaulted.
	Channel string
	// RequesterEmail is the reply address of an external requester. Nil for
	// internal tickets, which reach their requester through their user account.
	RequesterEmail *string
	// RequesterName is the display name of an external requester. It is
	// persisted ONLY when RequesterIsExternal is set -- an internal requester
	// is identified by requester_id and their name is resolved from the user
	// row on read, so a persisted copy would go stale at the first rename
	// (see the precedence note on ticketSelectColumns).
	RequesterName       *string
	RequesterIsExternal bool
	// CustomFields is a flat map of scalars. Nested objects, arrays and nulls
	// are rejected: the module renders one input per key and has no way to
	// display anything else.
	CustomFields map[string]any
}

// Intake limits. Generous enough that no honest form hits them, tight enough
// that a scripted submitter cannot grow a ticket row without bound -- the
// public intake paths (BACKLOG B8) write through here too.
const (
	maxRequesterEmailLen   = 320 // RFC 5321 local-part + "@" + domain
	maxRequesterNameLen    = 200
	maxCustomFieldKeys     = 100
	maxCustomFieldKeyLen   = 128
	maxCustomFieldValueLen = 4096
)

// normalize validates the intake and returns it in the form that goes to the
// database: channel defaulted, blank strings collapsed to nil, custom fields
// copied into a map that no caller still holds a reference to.
func (in TicketIntake) normalize() (TicketIntake, error) {
	out := TicketIntake{RequesterIsExternal: in.RequesterIsExternal}

	out.Channel = strings.TrimSpace(in.Channel)
	if out.Channel == "" {
		out.Channel = TicketChannelAgent
	}
	if !ValidTicketChannels[out.Channel] {
		return TicketIntake{}, ErrInvalidChannel
	}

	if in.RequesterEmail != nil {
		email := strings.TrimSpace(*in.RequesterEmail)
		if email != "" {
			if len(email) > maxRequesterEmailLen {
				return TicketIntake{}, ErrInvalidRequesterEmail
			}
			if _, err := mail.ParseAddress(email); err != nil {
				return TicketIntake{}, ErrInvalidRequesterEmail
			}
			out.RequesterEmail = &email
		}
	}

	// Only external requesters get a persisted name; for internal ones the
	// field is dropped here rather than at the call site, so every caller --
	// gRPC handler, form dispatch, future public intake -- gets the same rule.
	if in.RequesterIsExternal && in.RequesterName != nil {
		name := strings.TrimSpace(*in.RequesterName)
		if len([]rune(name)) > maxRequesterNameLen {
			return TicketIntake{}, ErrRequesterNameTooLong
		}
		if name != "" {
			out.RequesterName = &name
		}
	}

	fields, err := normalizeCustomFields(in.CustomFields)
	if err != nil {
		return TicketIntake{}, err
	}
	out.CustomFields = fields

	return out, nil
}

// normalizeCustomFields copies src into a fresh map, rejecting anything the
// module cannot render. It never returns nil: an absent map and an empty map
// mean the same thing here, and the column is NOT NULL DEFAULT '{}'.
func normalizeCustomFields(src map[string]any) (map[string]any, error) {
	out := make(map[string]any, len(src))
	if len(src) > maxCustomFieldKeys {
		return nil, ErrInvalidCustomFields
	}
	for key, value := range src {
		key = strings.TrimSpace(key)
		if key == "" || len(key) > maxCustomFieldKeyLen {
			return nil, ErrInvalidCustomFields
		}
		switch v := value.(type) {
		case string:
			if len(v) > maxCustomFieldValueLen {
				return nil, ErrInvalidCustomFields
			}
			out[key] = v
		case bool:
			out[key] = v
		case float64, float32, int, int32, int64:
			out[key] = v
		default:
			// nil, nested maps and arrays all land here.
			return nil, ErrInvalidCustomFields
		}
	}
	return out, nil
}
