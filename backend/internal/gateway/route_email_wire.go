package gateway

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/kmuhub/kmuhub/internal/server/response"
	"github.com/kmuhub/kmuhub/internal/validation"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// emailAddr is one recipient in a compose request. The desktop client sends
// {name,email}; older callers send a bare "a@b.de" string. Both are accepted.
type emailAddr struct {
	Name  string
	Email string
}

func (a *emailAddr) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) > 0 && b[0] == '"' {
		return json.Unmarshal(b, &a.Email)
	}
	var obj struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}
	if obj.Email == "" {
		return errors.New("address without email")
	}
	a.Name, a.Email = obj.Name, obj.Email
	return nil
}

type addrList []emailAddr

func (l addrList) proto() []*emailv1.EmailAddress {
	out := make([]*emailv1.EmailAddress, 0, len(l))
	for _, a := range l {
		out = append(out, &emailv1.EmailAddress{Name: a.Name, Email: a.Email})
	}
	return out
}

func (l addrList) emails() []string {
	out := make([]string, 0, len(l))
	for _, a := range l {
		out = append(out, a.Email)
	}
	return out
}

// validateRecipients checks every address with the same `dive,email` rule the
// plain-string DTOs used, so the error keeps its `to[0]` field path. It writes
// the 400 and returns false on failure; required-ness stays on the DTO tags.
func validateRecipients(w http.ResponseWriter, to, cc, bcc addrList) bool {
	body := struct {
		To  []string `json:"to" validate:"omitempty,dive,email"`
		Cc  []string `json:"cc" validate:"omitempty,dive,email"`
		Bcc []string `json:"bcc" validate:"omitempty,dive,email"`
	}{to.emails(), cc.emails(), bcc.emails()}
	err := validation.Validate(&body)
	if err == nil {
		return true
	}
	if errBody, ok := validation.ErrorBody(err); ok {
		response.JSON(w, http.StatusBadRequest, errBody)
	} else {
		response.Error(w, http.StatusBadRequest, "validation failed")
	}
	return false
}

// emailMessageListFields are the repeated fields of EmailMessageInfo. encoding/json
// drops an empty slice (`omitempty`), but the desktop client calls .map/.slice on
// every one of them, so each must always be written as an array.
var emailMessageListFields = []string{"references", "to", "cc", "bcc", "attachments", "label_ids"}

// writeEmailMessages writes an email RPC response that carries `message` and/or
// `messages` (EmailMessageInfo) with the list fields of every message, and the
// `messages` array itself, present even when empty. Encoding stays encoding/json
// so int64 fields (uid, size_bytes) remain numbers, as before.
func writeEmailMessages(w http.ResponseWriter, status int, resp any) {
	raw, err := json.Marshal(resp)
	if err != nil {
		slog.Error("email response marshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		slog.Error("email response unmarshal failed", "error", err)
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if msg, ok := top["message"]; ok {
		top["message"] = fillMessageLists(msg)
	}
	if msgs, ok := top["messages"]; ok {
		var items []json.RawMessage
		if json.Unmarshal(msgs, &items) == nil {
			for i := range items {
				items[i] = fillMessageLists(items[i])
			}
			top["messages"], _ = json.Marshal(items)
		}
	} else if _, isList := resp.(interface{ GetMessages() []*emailv1.EmailMessageInfo }); isList {
		top["messages"] = json.RawMessage("[]")
	}
	// Only the paged lists carry a total; omitempty drops it at 0.
	switch resp.(type) {
	case *emailv1.ListMessagesResponse, *emailv1.GetContactEmailsResponse:
		if _, ok := top["total"]; !ok {
			top["total"] = json.RawMessage("0")
		}
	}
	response.JSON(w, status, top)
}

// fillMessageLists adds `[]` for every list field missing from one message object.
func fillMessageLists(msg json.RawMessage) json.RawMessage {
	var obj map[string]json.RawMessage
	if json.Unmarshal(msg, &obj) != nil {
		return msg
	}
	if obj == nil {
		obj = map[string]json.RawMessage{}
	}
	for _, f := range emailMessageListFields {
		if _, ok := obj[f]; !ok {
			obj[f] = json.RawMessage("[]")
		}
	}
	out, err := json.Marshal(obj)
	if err != nil {
		return msg
	}
	return out
}
