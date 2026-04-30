package router

import (
	"context"
	"time"
)

func RunOpportunityLoop(
	ctx context.Context,
	store *BookStore,
	server *Server,
	markets []string,
	interval time.Duration,
) {
	if interval <= 0 {
		interval = 250 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			for _, market := range markets {
				book, ok := store.Get(market)
				if !ok {
					continue
				}

				server.PublishOpportunities(book)
			}
		}
	}
}
