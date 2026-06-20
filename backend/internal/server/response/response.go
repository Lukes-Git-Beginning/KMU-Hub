package response

import (
	"encoding/json"
	"log/slog"
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

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, errorResponse{Error: message})
}
