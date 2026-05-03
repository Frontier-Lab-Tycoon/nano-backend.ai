package db

import (
	"database/sql"
	"time"

	"github.com/seedspirit/nano-backend.ai/internal/common/encoding"
	"github.com/seedspirit/nano-backend.ai/internal/common/run"
)

func nullTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: encoding.FormatTime(*t), Valid: true}
}

func nullFailureReason(reason *run.FailureReason) sql.NullString {
	if reason == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(*reason), Valid: true}
}
