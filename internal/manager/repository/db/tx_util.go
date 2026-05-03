package db

import "github.com/jmoiron/sqlx"

func rollbackUnlessCommitted(tx *sqlx.Tx) {
	_ = tx.Rollback()
}
