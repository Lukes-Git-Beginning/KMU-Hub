package gateway

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"

	berichtev1 "github.com/kmuhub/kmuhub/proto/berichte/v1"
)

// Since 000367 only the create response has a secret to send; a listed link
// must not arrive with an empty `token` the desktop would build a URL from.
func TestToReportShareWire_TokenOnlyWhenMinted(t *testing.T) {
	listed, err := json.Marshal(toReportShareWire(&berichtev1.ShareToken{Id: "s1", CreatedAt: timestamppb.Now()}))
	if err != nil {
		t.Fatalf("marshal listed: %v", err)
	}
	if strings.Contains(string(listed), `"token"`) {
		t.Fatalf("listed link carries a token field: %s", listed)
	}

	minted, err := json.Marshal(toReportShareWire(&berichtev1.ShareToken{Id: "s1", Token: "secret", CreatedAt: timestamppb.Now()}))
	if err != nil {
		t.Fatalf("marshal minted: %v", err)
	}
	if !strings.Contains(string(minted), `"token":"secret"`) {
		t.Fatalf("minted link lost its token: %s", minted)
	}
}
