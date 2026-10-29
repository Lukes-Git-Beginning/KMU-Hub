package response

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"maps"
	"net/http"
	"regexp"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

// protoMarshalerFull is protoMarshaler with EmitUnpopulated: zero values, empty lists and
// unset messages (`null`) stay in the JSON. Frontends that test `=== null` or call
// `Number(x)` on a counter need the member to exist; protojson otherwise drops a 0 entirely.
// It still skips unset `optional` scalars, which fillOptionalNulls adds as `null`.
var protoMarshalerFull = protojson.MarshalOptions{
	UseProtoNames:   true,
	UseEnumNumbers:  true,
	EmitUnpopulated: true,
}

// taxModeNumber matches a numeric `"tax_mode":N` member. Only biz.v1.TaxMode fields are
// named tax_mode in emitted messages (the two string fields in biz.proto are request-only),
// and a key inside a JSON string value is escaped (`\"tax_mode\"`) so it cannot match.
var taxModeNumber = regexp.MustCompile(`("tax_mode":\s*)([123])`)

var taxModeNames = map[string]string{"1": "standard", "2": "reverse_charge", "3": "kleinunternehmer"}

// MarshalProto is the one place gateway handlers turn a proto message into JSON. It
// applies protoMarshaler and then writes biz.v1.TaxMode as the string the frontend's
// TaxMode type uses (`standard|reverse_charge|kleinunternehmer`) instead of the enum
// number, so an entity can be sent back unchanged in an update body.
// lean: key-based rewrite instead of per-enum marshaling (protojson has no such option);
// a second enum with the same wire need gets its own pattern or a protoreflect walk.
func MarshalProto(msg proto.Message) ([]byte, error) {
	return marshalProto(protoMarshaler, msg)
}

// MarshalProtoFull is MarshalProto with every field present (see protoMarshalerFull).
func MarshalProtoFull(msg proto.Message) ([]byte, error) {
	return marshalProto(protoMarshalerFull, msg)
}

func marshalProto(m protojson.MarshalOptions, msg proto.Message) ([]byte, error) {
	b, err := m.Marshal(msg)
	if err != nil {
		return nil, err
	}
	if m.EmitUnpopulated {
		if b, err = fillOptionalNulls(b, msg.ProtoReflect().Descriptor()); err != nil {
			return nil, err
		}
	}
	if bytes.Contains(b, []byte(`"tax_mode":`)) {
		b = taxModeNumber.ReplaceAllFunc(b, func(m []byte) []byte {
			sub := taxModeNumber.FindSubmatch(m)
			return append(append([]byte{}, sub[1]...), []byte(`"`+taxModeNames[string(sub[2])]+`"`)...)
		})
	}
	if names, ok := statusNames[msg.ProtoReflect().Descriptor().FullName()]; ok {
		return rewriteStatus(b, names)
	}
	return b, nil
}

// fillOptionalNulls adds `null` for every unset `optional` scalar of md and, recursively, of the
// messages nested in it, so "not set" (a position that was never counted) differs from a
// missing key. Map fields and well-known types (their JSON is not an object) stay untouched.
func fillOptionalNulls(b []byte, md protoreflect.MessageDescriptor) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, err
	}
	fields := md.Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		name := string(fd.Name())
		v, present := obj[name]
		switch {
		case !present:
			if fd.HasOptionalKeyword() && fd.Message() == nil {
				obj[name] = json.RawMessage("null")
			}
		case fd.Message() != nil && !fd.IsMap() && len(v) > 0 && (v[0] == '{' || v[0] == '['):
			if fd.IsList() {
				var elems []json.RawMessage
				if err := json.Unmarshal(v, &elems); err != nil {
					return nil, err
				}
				for j, e := range elems {
					if len(e) == 0 || e[0] != '{' {
						continue
					}
					filled, err := fillOptionalNulls(e, fd.Message())
					if err != nil {
						return nil, err
					}
					elems[j] = filled
				}
				filled, err := json.Marshal(elems)
				if err != nil {
					return nil, err
				}
				obj[name] = filled
			} else if v[0] == '{' {
				filled, err := fillOptionalNulls(v, fd.Message())
				if err != nil {
					return nil, err
				}
				obj[name] = filled
			}
		}
	}
	return json.Marshal(obj)
}

// statusNames lists the messages whose top-level `status` leaves as the string the frontend
// types it with (`draft|sent|paid`), not as the enum number. Keyed by message, not by field
// name: `status` is a member of dozens of other messages whose numbers stay numbers.
var statusNames = map[protoreflect.FullName]map[string]string{
	"hr.v1.WorkTimeEntry":  {"1": "active", "2": "completed", "3": "correction_pending", "4": "correction_approved", "5": "superseded"},
	"biz.v1.DunningRecord": {"1": "draft", "2": "sent", "3": "paid"},
	"biz.v1.CreditNote":    {"1": "draft", "2": "sent"},
	"video.v1.Recording":   {"1": "active", "2": "processing", "3": "completed", "4": "failed", "5": "deleted", "6": "pending"},
}

// rewriteStatus replaces a numeric top-level `status` member. An unknown number (a future
// enum value) stays a number rather than turning into a wrong word.
func rewriteStatus(b []byte, names map[string]string) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, err
	}
	if name, ok := names[string(obj["status"])]; ok {
		obj["status"] = json.RawMessage(`"` + name + `"`)
	}
	return json.Marshal(obj)
}

// Proto writes a protobuf message as JSON via protojson so well-known types
// (notably Timestamp) serialize correctly. Use this instead of JSON for handlers
// that return a proto.Message.
func Proto(w http.ResponseWriter, status int, msg proto.Message) {
	writeProto(w, status, MarshalProto, msg)
}

// ProtoFull is Proto with zero values, empty lists and unset optionals (`null`) kept in the
// JSON, for responses whose frontend type has no optional members.
func ProtoFull(w http.ResponseWriter, status int, msg proto.Message) {
	writeProto(w, status, MarshalProtoFull, msg)
}

func writeProto(w http.ResponseWriter, status int, marshal func(proto.Message) ([]byte, error), msg proto.Message) {
	b, err := marshal(msg)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

// ProtoWrapped writes msg under `key` (`{credit_note:{…}}`), the shape frontend hooks that
// `select` a single entity out of the response read.
func ProtoWrapped(w http.ResponseWriter, status int, key string, msg proto.Message) {
	b, err := MarshalProto(msg)
	if err != nil {
		slog.Error("proto json marshal failed", "error", err)
		Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	JSON(w, status, map[string]json.RawMessage{key: b})
}

// ProtoFlatten renders msg like Proto does, then lifts the fields of the nested
// message under `key` to the top level. The wire shape of wrapper messages such
// as ExpandedEventProto{event:{...}, calendar_name} becomes {...event fields...,
// calendar_name}, which is what flat frontend types read. Fields of the nested
// message win on a name clash.
func ProtoFlatten(msg proto.Message, key string) (json.RawMessage, error) {
	b, err := MarshalProto(msg)
	if err != nil {
		return nil, err
	}
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(b, &outer); err != nil {
		return nil, err
	}
	inner := map[string]json.RawMessage{}
	if raw, ok := outer[key]; ok {
		if err := json.Unmarshal(raw, &inner); err != nil {
			return nil, err
		}
		delete(outer, key)
	}
	for k, v := range outer {
		if _, taken := inner[k]; !taken {
			inner[k] = v
		}
	}
	return json.Marshal(inner)
}

// ProtoList writes a slice of proto messages as a bare JSON array via protojson,
// so a list endpoint returns `[item, ...]` (not a wrapped `{items:[...]}`) with the
// same snake_case + RFC3339 encoding as Proto. The frontend types these endpoints
// as T[]; an empty/nil slice serializes as `[]`.
func ProtoList[T proto.Message](w http.ResponseWriter, status int, msgs []T) {
	parts := make([]json.RawMessage, 0, len(msgs))
	for _, m := range msgs {
		b, err := MarshalProto(m)
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
		b, err := MarshalProto(m)
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
