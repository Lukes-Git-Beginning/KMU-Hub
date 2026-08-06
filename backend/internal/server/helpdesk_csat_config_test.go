package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/kmuhub/kmuhub/internal/helpdesk"
	settingsv1 "github.com/kmuhub/kmuhub/proto/settings/v1"
)

func TestCsatConfigFromEntries_EmptyReturnsDefault(t *testing.T) {
	got := csatConfigFromEntries(nil)
	assert.Equal(t, helpdesk.DefaultCsatConfig(), got)
}

func TestCsatConfigFromEntries_IgnoresOtherKeys(t *testing.T) {
	value, err := structpb.NewValue(map[string]any{"enabled": false})
	require.NoError(t, err)
	got := csatConfigFromEntries([]*settingsv1.SettingEntry{{Key: "not-the-config-key", Value: value}})
	assert.Equal(t, helpdesk.DefaultCsatConfig(), got)
}

func TestCsatConfigRoundTrip(t *testing.T) {
	cfg := helpdesk.CsatConfig{
		Enabled:            false,
		SurveyDelayMinutes: 90,
		SurveyQuestion:     "Wie war's?",
	}

	value, err := csatConfigToValue(cfg)
	require.NoError(t, err)

	got := csatConfigFromEntries([]*settingsv1.SettingEntry{{Key: helpdesk.CsatConfigEntryKey, Value: value}})
	assert.Equal(t, cfg, got)
}

func TestCsatConfigFromEntries_MalformedFieldsFallBackToDefault(t *testing.T) {
	// enabled is a string instead of a bool, delay_minutes is missing --
	// both fields must fall back to the default rather than zero-value or error.
	value, err := structpb.NewValue(map[string]any{
		"enabled":  "yes",
		"question": "custom question",
	})
	require.NoError(t, err)

	got := csatConfigFromEntries([]*settingsv1.SettingEntry{{Key: helpdesk.CsatConfigEntryKey, Value: value}})

	want := helpdesk.DefaultCsatConfig()
	want.SurveyQuestion = "custom question"
	assert.Equal(t, want, got)
}
