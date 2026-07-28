package response

import (
	"encoding/json"
	"log/slog"
	"maps"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type errorResponse struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// protoMarshaler serializes proto messages with options chosen to match the wire
// shape the frontend already expects from encoding/json, while fixing the one
// thing encoding/json gets wrong: google.protobuf.Timestamp (and other
// well-known types) serialize as proper RFC3339 strings instead of
// {seconds,nanos}. R3-P0-1.
//   - UseProtoNames  → snake_case field names (matches the pb.go `json:"..."` tags)
//   - UseEnumNumbers → integer enum values (the frontend types enums as number)
//   - EmitUnpopulated stays false → omitempty-like behavior preserved
//
// NOTE: protojson emits int64/uint64 as JSON strings (proto3 JSON spec). Migrate
// handlers to Proto() per module and verify the frontend reads any 64-bit fields
// as strings before switching them over.
var protoMarshaler = protojson.MarshalOptions{
	UseProtoNames:  true,
	UseEnumNumbers: true,
}

// Proto writes a protobuf message as JSON via protojson so well-known types
// (notably Timestamp) serialize correctly. Use this instead of JSON for handlers
// that return a proto.Message.
func Proto(w http.ResponseWriter, status int, msg proto.Message) {
	b, err := protoMarshaler.Marshal(msg)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

// ProtoList writes a slice of proto messages as a bare JSON array via protojson,
// so a list endpoint returns `[item, ...]` (not a wrapped `{items:[...]}`) with the
// same snake_case + RFC3339 encoding as Proto. The frontend types these endpoints
// as T[]; an empty/nil slice serializes as `[]`.
func ProtoList[T proto.Message](w http.ResponseWriter, status int, msgs []T) {
	parts := make([]json.RawMessage, 0, len(msgs))
	for _, m := range msgs {
		b, err := protoMarshaler.Marshal(m)
		if err != nil {
			slog.Error("proto json marshal failed", "error", err)
			Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		parts = append(parts, b)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(parts)
}

// ProtoListWrapped writes a slice of proto messages under `key` in a JSON
// object, with the same protojson per-item encoding as ProtoList (so
// Timestamps/enums stay consistent), plus any additional top-level fields.
// Use this instead of ProtoList when the frontend type expects a wrapped
// list (e.g. `{folders:[...], total}`) rather than a bare array.
func ProtoListWrapped[T proto.Message](w http.ResponseWriter, status int, key string, msgs []T, extra map[string]any) {
	parts := make([]json.RawMessage, 0, len(msgs))
	for _, m := range msgs {
		b, err := protoMarshaler.Marshal(m)
		if err != nil {
			slog.Error("proto json marshal failed", "error", err)
			Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		parts = append(parts, b)
	}
	body := make(map[string]any, len(extra)+1)
	maps.Copy(body, extra)
	body[key] = parts
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, errorResponse{Error: message})
}
