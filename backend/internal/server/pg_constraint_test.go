package server

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kmuhub/kmuhub/internal/vermietung"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPgConstraintError_Codes(t *testing.T) {
	cases := []struct {
		name string
		pg   *pgconn.PgError
		code codes.Code
	}{
		{"check violation", &pgconn.PgError{Code: "23514"}, codes.InvalidArgument},
		{"invalid text representation", &pgconn.PgError{Code: "22P02"}, codes.InvalidArgument},
		{"not null violation", &pgconn.PgError{Code: "23502"}, codes.InvalidArgument},
		{"value too long", &pgconn.PgError{Code: "22001"}, codes.InvalidArgument},
		{"numeric out of range", &pgconn.PgError{Code: "22003"}, codes.InvalidArgument},
		{"unknown reference on insert", &pgconn.PgError{Code: "23503", Detail: `Key (x)=(1) is not present in table "y".`}, codes.InvalidArgument},
		{"still referenced on delete", &pgconn.PgError{Code: "23503", Detail: `Key (x)=(1) is still referenced from table "y".`}, codes.FailedPrecondition},
		{"unique violation", &pgconn.PgError{Code: "23505"}, codes.AlreadyExists},
		{"exclusion violation", &pgconn.PgError{Code: "23P01"}, codes.AlreadyExists},
		{"deadlock detected", &pgconn.PgError{Code: "40P01"}, codes.Aborted},
		{"serialization failure", &pgconn.PgError{Code: "40001"}, codes.Aborted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.pg.Message = "secret message"
			tc.pg.TableName = "secret_table"
			tc.pg.ConstraintName = "secret_constraint"
			got := pgConstraintError(fmt.Errorf("insert: %w", tc.pg))
			require.Error(t, got)
			st, ok := status.FromError(got)
			require.True(t, ok)
			require.Equal(t, tc.code, st.Code())
			require.NotContains(t, st.Message(), "secret")
		})
	}

	t.Run("other sqlstate", func(t *testing.T) {
		require.NoError(t, pgConstraintError(&pgconn.PgError{Code: "42P01"}))
	})
	t.Run("no pg error", func(t *testing.T) {
		require.NoError(t, pgConstraintError(errors.New("boom")))
	})
}

// TestMapErrors_PostgresConstraintNotInternal runs every map*Error function
// with a wrapped FK, CHECK and enum violation. Before the shared default-branch
// hook all of them answered Internal (HTTP 500).
func TestMapErrors_PostgresConstraintNotInternal(t *testing.T) {
	mappers := map[string]func(error) error{
		"auth": mapError, "automation": mapDomainError, "berichte": mapBerichteError,
		"bexio": mapBexioError, "biz": mapBizError, "bankaccount": bankAccountError,
		"banking": bankingError, "einvoice": mapEInvoiceError, "expense": expenseError,
		"gobd": mapGobdArchiveError, "calendar": mapCalendarError, "chat": mapChatError,
		"crm": mapCRMError, "datev": mapDatevUploadError, "dialer": mapDialerError,
		"document": mapDocumentError, "einkauf": mapEinkaufError, "email": mapEmailError,
		"formulare": mapFormulareError, "fuhrpark": mapFuhrparkError, "helpdesk": mapHelpdeskError,
		"hr": mapHRError, "inbox": mapInboxError, "inventar": mapInventarError,
		"lexware": mapLexwareError, "notification": mapNotificationError, "plugin": mapPluginError,
		"produktion": mapProduktionError, "produktion-ext": mapProduktionExtError,
		"rapporte": mapRapporteError, "schichten": mapSchichtenError, "security": mapSecurityError,
		"settings":   func(err error) error { return mapModuleGrantError(err, "grant") },
		"vermietung": mapVermietungError, "vertraege": mapVertraegeError, "video": mapVideoError,
		"recording": mapRecordingError, "meeting": mapMeetingError, "presence": mapPresenceError,
		"wiki": mapWikiError, "timeentry": mapTimeEntryError, "work": mapWorkError,
	}
	violations := map[string]codes.Code{
		"23503": codes.InvalidArgument, "23514": codes.InvalidArgument, "22P02": codes.InvalidArgument,
		"23P01": codes.AlreadyExists, "40P01": codes.Aborted,
	}
	for name, fn := range mappers {
		for sqlstate, want := range violations {
			t.Run(name+"/"+sqlstate, func(t *testing.T) {
				err := fn(fmt.Errorf("repo: %w", &pgconn.PgError{Code: sqlstate}))
				require.Equal(t, want, status.Code(err))
			})
		}
	}
}

// TestMapErrors_DomainConflictBeatsPgBackstop: a domain error that wraps an
// exclusion violation keeps its own mapping, the backstop only fires in default.
func TestMapErrors_DomainConflictBeatsPgBackstop(t *testing.T) {
	err := mapVermietungError(fmt.Errorf("%w: %w", vermietung.ErrRentalConflict, &pgconn.PgError{Code: "23P01"}))
	require.Equal(t, codes.AlreadyExists, status.Code(err))
	require.NotContains(t, status.Convert(err).Message(), "overlaps with an existing record")
}
