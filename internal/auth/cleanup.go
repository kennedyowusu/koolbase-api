package auth

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

func StartCleanupJob(repo Repository) {
	go func() {
		for {
			// Run at midnight
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			time.Sleep(time.Until(next))

			ctx := context.Background()
			if err := repo.PurgeDeletedAccounts(ctx); err != nil {
				log.Error().Err(err).Msg("purge deleted accounts failed")
			} else {
				log.Info().Msg("purged accounts deleted more than 30 days ago")
			}
		}
	}()
}
