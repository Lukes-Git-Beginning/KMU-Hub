package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"google.golang.org/protobuf/proto"

	"github.com/kmuhub/kmuhub/internal/server/response"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

// The desktop inbox types are flat and string-typed: tags is always an array,
// team enums are strings ('round_robin', 'admin'), lists are bare arrays and
// objects are bare. The proto responses are wrapped, number-typed and drop empty
// repeated fields, so every inbox handler shapes its body through the helpers
// below instead of response.Proto.

type jsonObject = map[string]json.RawMessage

// protoObject renders msg like response.Proto and returns it as a mutable object.
func protoObject(msg proto.Message) (jsonObject, error) {
	b, err := cannedResponseMarshaler.Marshal(msg)
	if err != nil {
		return nil, err
	}
	obj := jsonObject{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func setJSON(obj jsonObject, key string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	obj[key] = b
}

func setDefault(obj jsonObject, key string, raw string) {
	if _, ok := obj[key]; !ok {
		obj[key] = json.RawMessage(raw)
	}
}

func channelName(c inboxv1.Channel) string {
	switch c {
	case inboxv1.Channel_CHANNEL_EMAIL:
		return "email"
	case inboxv1.Channel_CHANNEL_CHAT:
		return "chat"
	case inboxv1.Channel_CHANNEL_NOTIFICATION:
		return "notification"
	case inboxv1.Channel_CHANNEL_GUEST:
		return "guest"
	default:
		return ""
	}
}

func assignmentModeName(m inboxv1.AssignmentMode) string {
	if m == inboxv1.AssignmentMode_ASSIGNMENT_MODE_ROUND_ROBIN {
		return "round_robin"
	}
	return "manual"
}

func visibilityName(v inboxv1.TeamInboxVisibility) string {
	if v == inboxv1.TeamInboxVisibility_VISIBILITY_PRIVATE {
		return "private"
	}
	return "open"
}

func teamRoleName(r inboxv1.TeamMemberRole) string {
	if r == inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN {
		return "admin"
	}
	return "member"
}

// inboxMessageObject: tags is always an array and the flags are always present.
// The channel stays a number — the desktop client maps it itself.
func inboxMessageObject(m *inboxv1.InboxMessageInfo) (jsonObject, error) {
	obj, err := protoObject(m)
	if err != nil {
		return nil, err
	}
	setDefault(obj, "tags", "[]")
	setDefault(obj, "is_read", "false")
	setDefault(obj, "is_starred", "false")
	setDefault(obj, "is_archived", "false")
	return obj, nil
}

func inboxTeamObject(t *inboxv1.TeamInboxInfo) (jsonObject, error) {
	obj, err := protoObject(t)
	if err != nil {
		return nil, err
	}
	setJSON(obj, "assignment_mode", assignmentModeName(t.GetAssignmentMode()))
	setJSON(obj, "visibility", visibilityName(t.GetVisibility()))
	return obj, nil
}

func inboxMemberObject(m *inboxv1.TeamMemberInfo) (jsonObject, error) {
	obj, err := protoObject(m)
	if err != nil {
		return nil, err
	}
	setJSON(obj, "role", teamRoleName(m.GetRole()))
	return obj, nil
}

// inboxRuleObject: channel is a string (or absent), is_active/priority always present.
func inboxRuleObject(rule *inboxv1.RoutingRuleInfo) (jsonObject, error) {
	obj, err := protoObject(rule)
	if err != nil {
		return nil, err
	}
	if rule.Channel != nil {
		setJSON(obj, "channel", channelName(rule.GetChannel()))
	}
	setDefault(obj, "is_active", "false")
	setDefault(obj, "priority", "0")
	return obj, nil
}

// writeInboxList shapes every item with fn and writes them as one bare array.
func writeInboxList[T proto.Message](w http.ResponseWriter, status int, items []T, fn func(T) (jsonObject, error)) {
	out := make([]jsonObject, 0, len(items))
	for _, it := range items {
		obj, err := fn(it)
		if err != nil {
			writeInboxMarshalError(w, err)
			return
		}
		out = append(out, obj)
	}
	response.JSON(w, status, out)
}

func writeInboxObject(w http.ResponseWriter, status int, obj jsonObject, err error) {
	if err != nil {
		writeInboxMarshalError(w, err)
		return
	}
	response.JSON(w, status, obj)
}

func writeInboxMarshalError(w http.ResponseWriter, err error) {
	slog.Error("inbox response marshal failed", "error", err)
	response.Error(w, http.StatusInternalServerError, "internal server error")
}

// channelFromJSON reads a channel given as a name ("email") or a number (1).
func channelFromJSON(raw json.RawMessage) inboxv1.Channel {
	var name string
	if json.Unmarshal(raw, &name) == nil {
		return parseChannelQuery(name)
	}
	var n int32
	if json.Unmarshal(raw, &n) == nil {
		if _, known := inboxv1.Channel_name[n]; known {
			return inboxv1.Channel(n)
		}
	}
	return inboxv1.Channel_CHANNEL_UNSPECIFIED
}

// ruleTestMessage is the sample message of POST /rules/test. The desktop client
// sends it as `message` with the channel as a string, which the proto-typed
// decoding of `test_message` could not read.
type ruleTestMessage struct {
	Channel     json.RawMessage `json:"channel"`
	SenderName  string          `json:"sender_name"`
	SenderEmail *string         `json:"sender_email"`
	Subject     string          `json:"subject"`
	Preview     string          `json:"preview"`
	Tags        []string        `json:"tags"`
}

func (m *ruleTestMessage) proto() *inboxv1.InboxMessageInfo {
	if m == nil {
		return nil
	}
	return &inboxv1.InboxMessageInfo{
		Channel:     channelFromJSON(m.Channel),
		SenderName:  m.SenderName,
		SenderEmail: m.SenderEmail,
		Subject:     m.Subject,
		Preview:     m.Preview,
		Tags:        m.Tags,
	}
}
