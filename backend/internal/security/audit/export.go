package audit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kmuhub/kmuhub/internal/models"
)

// UTF-8 BOM for Excel compatibility
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ExportCSV exports audit entries as RFC 4180 CSV with UTF-8 BOM.
func ExportCSV(entries []*models.AuditEntry) ([]byte, error) {
	var buf bytes.Buffer

	// Write UTF-8 BOM for Excel compatibility
	buf.Write(utf8BOM)

	w := csv.NewWriter(&buf)

	// Write header row
	header := []string{
		"ID", "Sequence", "Timestamp", "User ID", "Action",
		"Target", "Target Type", "Details", "IP Address",
		"User Agent", "Result", "Entry Hash",
	}
	if err := w.Write(header); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}

	// Write data rows
	for _, e := range entries {
		userID := ""
		if e.UserID != nil {
			userID = e.UserID.String()
		}

		row := []string{
			e.ID.String(),
			fmt.Sprintf("%d", e.SequenceNum),
			e.Timestamp.UTC().Format(time.RFC3339),
			userID,
			e.Action,
			e.Target,
			e.TargetType,
			e.Details,
			e.IPAddress,
			e.UserAgent,
			e.Result,
			e.EntryHash,
		}
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv flush: %w", err)
	}

	return buf.Bytes(), nil
}

// auditEntryExport is the JSON export representation of an audit entry.
type auditEntryExport struct {
	ID          string `json:"id"`
	SequenceNum int64  `json:"sequence_num"`
	Timestamp   string `json:"timestamp"`
	UserID      string `json:"user_id,omitempty"`
	Action      string `json:"action"`
	Target      string `json:"target,omitempty"`
	TargetType  string `json:"target_type,omitempty"`
	Details     string `json:"details,omitempty"`
	IPAddress   string `json:"ip_address,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	Result      string `json:"result"`
	EntryHash   string `json:"entry_hash"`
}

// ExportJSON exports audit entries as an indented JSON array.
func ExportJSON(entries []*models.AuditEntry) ([]byte, error) {
	exports := make([]auditEntryExport, 0, len(entries))

	for _, e := range entries {
		export := auditEntryExport{
			ID:          e.ID.String(),
			SequenceNum: e.SequenceNum,
			Timestamp:   e.Timestamp.UTC().Format(time.RFC3339),
			Action:      e.Action,
			Target:      e.Target,
			TargetType:  e.TargetType,
			Details:     e.Details,
			IPAddress:   e.IPAddress,
			UserAgent:   e.UserAgent,
			Result:      e.Result,
			EntryHash:   e.EntryHash,
		}
		if e.UserID != nil {
			export.UserID = e.UserID.String()
		}
		exports = append(exports, export)
	}

	data, err := json.MarshalIndent(exports, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal json: %w", err)
	}

	return data, nil
}
