package gateway

// Wire shape of the Berichte definition, schedule and run routes as the frontend types them
// (berichte-types.ts: ReportDefinition, ReportSchedule, ReportRun, ReportResult). The `bytes`
// proto fields (query_config, params, payload) are raw JSON, not base64 strings: the builder
// reads `query_config` as an object and every chart reads `result.rows`/`result.columns`
// directly. The gRPC service keeps its proto types; the conversion sits at this edge only.

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/kmuhub/kmuhub/internal/server/response"
	berichtev1 "github.com/kmuhub/kmuhub/proto/berichte/v1"
)

type reportDefinitionWire struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Module        string          `json:"module"`
	Kind          string          `json:"kind"`
	QueryConfig   json.RawMessage `json:"query_config"`
	DefaultFormat string          `json:"default_format"`
	CreatedBy     *string         `json:"created_by"`
	IsPublished   bool            `json:"is_published"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func toReportDefinitionWire(d *berichtev1.Definition) reportDefinitionWire {
	return reportDefinitionWire{
		ID:            d.GetId(),
		TenantID:      d.GetTenantId(),
		Name:          d.GetName(),
		Description:   d.GetDescription(),
		Module:        d.GetModule(),
		Kind:          d.GetKind(),
		QueryConfig:   jsonOrDefault(d.GetQueryConfig(), "{}"),
		DefaultFormat: d.GetDefaultFormat(),
		CreatedBy:     d.CreatedBy,
		IsPublished:   d.GetIsPublished(),
		CreatedAt:     d.GetCreatedAt().AsTime(),
		UpdatedAt:     d.GetUpdatedAt().AsTime(),
	}
}

type reportDefinitionEnvelope struct {
	Definition reportDefinitionWire `json:"definition"`
}

type reportDefinitionListWire struct {
	Definitions []reportDefinitionWire `json:"definitions"`
	Total       int32                  `json:"total"`
}

func toReportDefinitionListWire(resp *berichtev1.ListDefinitionsResponse) reportDefinitionListWire {
	defs := make([]reportDefinitionWire, 0, len(resp.GetDefinitions()))
	for _, d := range resp.GetDefinitions() {
		defs = append(defs, toReportDefinitionWire(d))
	}
	return reportDefinitionListWire{Definitions: defs, Total: resp.GetTotal()}
}

type reportScheduleWire struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	DefinitionID   string          `json:"definition_id"`
	Name           string          `json:"name"`
	CronExpression string          `json:"cron_expression"`
	Recipients     []string        `json:"recipients"`
	Format         string          `json:"format"`
	Params         json.RawMessage `json:"params"`
	Active         bool            `json:"active"`
	LastRunAt      *time.Time      `json:"last_run_at"`
	LastRunStatus  *string         `json:"last_run_status"`
	LastRunError   *string         `json:"last_run_error"`
	CreatedBy      *string         `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func toReportScheduleWire(s *berichtev1.Schedule) reportScheduleWire {
	recipients := s.GetRecipients()
	if recipients == nil {
		recipients = []string{}
	}
	return reportScheduleWire{
		ID:             s.GetId(),
		TenantID:       s.GetTenantId(),
		DefinitionID:   s.GetDefinitionId(),
		Name:           s.GetName(),
		CronExpression: s.GetCronExpression(),
		Recipients:     recipients,
		Format:         s.GetFormat(),
		Params:         jsonOrDefault(s.GetParams(), "{}"),
		Active:         s.GetActive(),
		LastRunAt:      timePtr(s.GetLastRunAt()),
		LastRunStatus:  s.LastRunStatus,
		LastRunError:   s.LastRunError,
		CreatedBy:      s.CreatedBy,
		CreatedAt:      s.GetCreatedAt().AsTime(),
		UpdatedAt:      s.GetUpdatedAt().AsTime(),
	}
}

type reportScheduleEnvelope struct {
	Schedule reportScheduleWire `json:"schedule"`
}

type reportScheduleListWire struct {
	Schedules []reportScheduleWire `json:"schedules"`
	Total     int32                `json:"total"`
}

func toReportScheduleListWire(resp *berichtev1.ListSchedulesResponse) reportScheduleListWire {
	schedules := make([]reportScheduleWire, 0, len(resp.GetSchedules()))
	for _, s := range resp.GetSchedules() {
		schedules = append(schedules, toReportScheduleWire(s))
	}
	return reportScheduleListWire{Schedules: schedules, Total: resp.GetTotal()}
}

type reportRunWire struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	DefinitionID string          `json:"definition_id"`
	ScheduleID   *string         `json:"schedule_id"`
	Trigger      string          `json:"trigger"`
	Params       json.RawMessage `json:"params"`
	DurationMs   *int32          `json:"duration_ms"`
	RowCount     *int32          `json:"row_count"`
	Status       string          `json:"status"`
	Error        *string         `json:"error"`
	StartedAt    time.Time       `json:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at"`
}

func toReportRunWire(r *berichtev1.Run) *reportRunWire {
	if r == nil {
		return nil
	}
	return &reportRunWire{
		ID:           r.GetId(),
		TenantID:     r.GetTenantId(),
		DefinitionID: r.GetDefinitionId(),
		ScheduleID:   r.ScheduleId,
		Trigger:      r.GetTrigger(),
		Params:       jsonOrDefault(r.GetParams(), "{}"),
		DurationMs:   r.DurationMs,
		RowCount:     r.RowCount,
		Status:       r.GetStatus(),
		Error:        r.Error,
		StartedAt:    r.GetStartedAt().AsTime(),
		CompletedAt:  timePtr(r.GetCompletedAt()),
	}
}

// payloadColumn and payloadSeries are the shapes the executors serialise into the result payload
// (berichte.Column, berichte.Series); the FE reads `key`/`type` and `id`/`data` instead.
type payloadColumn struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

type reportColumnWire struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type,omitempty"`
}

type payloadSeries struct {
	Label      string `json:"label"`
	DataPoints []struct {
		Label string  `json:"label"`
		Value float64 `json:"value"`
	} `json:"data_points"`
}

type reportSeriesPointWire struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type reportSeriesWire struct {
	ID    string                  `json:"id"`
	Label string                  `json:"label"`
	Data  []reportSeriesPointWire `json:"data"`
}

// toReportResultWire turns the proto ReportResult into the FE's ReportResult: the JSON payload
// ({columns, rows, series, totals, meta}) lifted to the top level, columns and series renamed to
// the FE keys, and meta completed from the proto fields (generated_at, row_count, from_cache).
// A payload that is not a JSON object is an error, not a silent `null`.
func toReportResultWire(res *berichtev1.ReportResult, definitionID string) (map[string]json.RawMessage, error) {
	obj := map[string]json.RawMessage{}
	if payload := res.GetPayload(); len(payload) > 0 {
		if err := json.Unmarshal(payload, &obj); err != nil {
			return nil, err
		}
	}
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}

	var cols []payloadColumn
	if raw, ok := obj["columns"]; ok {
		if err := json.Unmarshal(raw, &cols); err != nil {
			return nil, err
		}
	}
	colsWire := make([]reportColumnWire, 0, len(cols))
	for _, c := range cols {
		colsWire = append(colsWire, reportColumnWire{Key: c.Name, Label: c.Label, Type: c.Kind})
	}

	var series []payloadSeries
	if raw, ok := obj["series"]; ok {
		if err := json.Unmarshal(raw, &series); err != nil {
			return nil, err
		}
	}
	seriesWire := make([]reportSeriesWire, 0, len(series))
	for _, s := range series {
		points := make([]reportSeriesPointWire, 0, len(s.DataPoints))
		for _, p := range s.DataPoints {
			points = append(points, reportSeriesPointWire{Label: p.Label, Value: p.Value})
		}
		seriesWire = append(seriesWire, reportSeriesWire{ID: s.Label, Label: s.Label, Data: points})
	}

	meta := map[string]json.RawMessage{}
	if raw, ok := obj["meta"]; ok {
		if err := json.Unmarshal(raw, &meta); err != nil {
			return nil, err
		}
		if meta == nil {
			meta = map[string]json.RawMessage{}
		}
	}
	set := func(key string, v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		meta[key] = b
		return nil
	}
	if res.GetGeneratedAt() != nil {
		if err := set("generated_at", res.GetGeneratedAt().AsTime()); err != nil {
			return nil, err
		}
	}
	if err := set("row_count", res.GetRowCount()); err != nil {
		return nil, err
	}
	if err := set("from_cache", res.GetFromCache()); err != nil {
		return nil, err
	}
	if _, ok := meta["definition_id"]; !ok && definitionID != "" {
		if err := set("definition_id", definitionID); err != nil {
			return nil, err
		}
	}

	for key, v := range map[string]any{"columns": colsWire, "series": seriesWire, "meta": meta} {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		obj[key] = b
	}
	if raw, ok := obj["rows"]; !ok || string(raw) == "null" {
		obj["rows"] = json.RawMessage("[]")
	}
	return obj, nil
}

type reportRunResponseWire struct {
	Result map[string]json.RawMessage `json:"result"`
	Run    *reportRunWire             `json:"run"`
}

func writeReportRun(w http.ResponseWriter, resp *berichtev1.RunReportResponse) {
	result, err := toReportResultWire(resp.GetResult(), resp.GetRun().GetDefinitionId())
	if err != nil {
		slog.Error("berichte: run result payload is not a report result", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.JSON(w, http.StatusOK, reportRunResponseWire{Result: result, Run: toReportRunWire(resp.GetRun())})
}
