package helpdesk

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultCsatConfig_IsValid(t *testing.T) {
	if err := ValidateCsatConfig(DefaultCsatConfig()); err != nil {
		t.Fatalf("DefaultCsatConfig must be valid, got %v", err)
	}
}

// TestDefaultCsatConfig_IsOptIn pins the default down deliberately. A tenant
// that has never written a config must not have surveys mailed out on its
// behalf: the invitation links to CSAT_SURVEY_BASE_URL, where nothing is
// served, and there is no exposed route through which a tenant could switch
// them back off. Flipping this to true needs both of those fixed first --
// the reasoning is spelled out on DefaultCsatConfig itself.
func TestDefaultCsatConfig_IsOptIn(t *testing.T) {
	if DefaultCsatConfig().Enabled {
		t.Fatal("CSAT surveys must be opt-in: a tenant that never configured them must not send any")
	}
}

func TestValidateCsatConfig_DelayBounds(t *testing.T) {
	cases := []struct {
		name    string
		delay   int32
		wantErr error
	}{
		{"zero is valid", 0, nil},
		{"max is valid", csatSurveyDelayMaxMinutes, nil},
		{"negative rejected", -1, ErrInvalidCsatDelay},
		{"over max rejected", csatSurveyDelayMaxMinutes + 1, ErrInvalidCsatDelay},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := CsatConfig{Enabled: true, SurveyDelayMinutes: tc.delay, SurveyQuestion: "q"}
			err := ValidateCsatConfig(cfg)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("delay %d: got err %v, want %v", tc.delay, err, tc.wantErr)
			}
		})
	}
}

func TestValidateCsatConfig_QuestionTooLong(t *testing.T) {
	cfg := CsatConfig{Enabled: true, SurveyDelayMinutes: 60, SurveyQuestion: strings.Repeat("x", 501)}
	if err := ValidateCsatConfig(cfg); !errors.Is(err, ErrCsatQuestionTooLong) {
		t.Errorf("got %v, want ErrCsatQuestionTooLong", err)
	}

	cfg.SurveyQuestion = strings.Repeat("x", 500)
	if err := ValidateCsatConfig(cfg); err != nil {
		t.Errorf("500 chars must be valid, got %v", err)
	}
}
