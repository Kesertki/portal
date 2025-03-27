package handlers

import (
	"database/sql"

	"github.com/rs/zerolog/log"
)

func rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil {
		log.Error().Err(err).Msg("Failed to rollback transaction")
	}
}
