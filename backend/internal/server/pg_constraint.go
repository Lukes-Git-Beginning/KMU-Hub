package server

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// pgConstraintError maps a Postgres data error found anywhere in err's chain to
// a client status. It is the first line of every map*Error default branch, so
// a CHECK, FK or type violation the service layer did not validate answers
// 400/409 instead of falling through to Internal (500).
//
// The text sent to the client is fixed. pgErr.Message, TableName and
// ConstraintName stay in the log only: they name tables and columns.
// Returns nil when err carries no PgError or its class is not a data error.
func pgConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	var st error
	switch pgErr.Code {
	case "23514", "22P02", "23502", "22001", "22003", "22007", "22008":
		// check_violation, invalid_text_representation (bad enum/uuid literal),
		// not_null_violation, string_data_right_truncation, numeric_value_out_of_range,
		// invalid_datetime_format, datetime_field_overflow.
		st = status.Error(codes.InvalidArgument, "invalid value in request")
	case "23503":
		// lean: Insert und Delete werden am (englischen) Detail-Text getrennt; bei
		// anderer lc_messages fällt jede FK-Verletzung auf InvalidArgument zurück.
		if strings.Contains(pgErr.Detail, "still referenced") {
			st = status.Error(codes.FailedPrecondition, "record is still in use")
		} else {
			st = status.Error(codes.InvalidArgument, "referenced record does not exist")
		}
	case "23505":
		st = status.Error(codes.AlreadyExists, "record already exists")
	case "23P01":
		// exclusion_violation: overlapping booking/rental window.
		st = status.Error(codes.AlreadyExists, "overlaps with an existing record")
	case "40P01", "40001":
		// deadlock_detected, serialization_failure: the statement lost against a
		// concurrent writer and is safe to retry. Unavailable (503), not Aborted (409):
		// the idempotency store keeps every answer below 500 for 24 h and replays it for
		// the same key, so a 409 would turn one transient deadlock into a permanent one
		// for clients that retry with a stable key (desktop offline queue).
		st = status.Error(codes.Unavailable, "concurrent change, please retry")
	default:
		return nil
	}
	slog.Warn("postgres constraint violation mapped to client error",
		"sqlstate", pgErr.Code, "constraint", pgErr.ConstraintName, "table", pgErr.TableName, "error", err)
	return st
}
