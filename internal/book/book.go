package book

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

var ErrEmptyBook = errors.New("book has no valid top of book")

type Book struct {
	mu sync.RWMutex

	venue             domain.Venue
	venueMarketID     string
	canonicalMarketID string

	yesBids map[domain.PriceBps]int64
	yesAsks map[domain.PriceBps]int64

	lastSequence uint64
	lastUpdate   time.Time
	staleAfter   time.Duration
}

func NewBook(
	venue domain.Venue,
	venueMarketID string,
	canonicalMarketID string,
	staleAfter time.Duration,
) *Book {
	return &Book{
		venue:             venue,
		venueMarketID:     venueMarketID,
		canonicalMarketID: canonicalMarketID,
		yesBids:           make(map[domain.PriceBps]int64),
		yesAsks:           make(map[domain.PriceBps]int64),
		staleAfter:        staleAfter,
	}
}

func (b *Book) ApplySnapshot(
	yesBids []domain.PriceLevel,
	yesAsks []domain.PriceLevel,
	sequence uint64,
	receiveTs time.Time,
) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.yesBids = make(map[domain.PriceBps]int64, len(yesBids))
	b.yesAsks = make(map[domain.PriceBps]int64, len(yesAsks))

	for _, level := range yesBids {
		if level.Quantity > 0 {
			b.yesBids[level.PriceBps] = level.Quantity
		}
	}

	for _, level := range yesAsks {
		if level.Quantity > 0 {
			b.yesAsks[level.PriceBps] = level.Quantity
		}
	}

	b.lastSequence = sequence
	b.lastUpdate = receiveTs
}

func (b *Book) ApplyDelta(
	isBid bool,
	price domain.PriceBps,
	quantity int64,
	sequence uint64,
	receiveTs time.Time,
) {
	b.mu.Lock()
	defer b.mu.Unlock()

	target := b.yesAsks
	if isBid {
		target = b.yesBids
	}

	if quantity <= 0 {
		delete(target, price)
	} else {
		target[price] = quantity
	}

	b.lastSequence = sequence
	b.lastUpdate = receiveTs
}

func (b *Book) TopOfBook() (domain.TopOfBook, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	bestBidPrice, bestBidQty, hasBid := bestBid(b.yesBids)
	bestAskPrice, bestAskQty, hasAsk := bestAsk(b.yesAsks)

	if !hasBid && !hasAsk {
		return domain.TopOfBook{}, ErrEmptyBook
	}

	return domain.TopOfBook{
		Venue:             b.venue,
		VenueMarketID:     b.venueMarketID,
		CanonicalMarketID: b.canonicalMarketID,
		YesBidPriceBps:    bestBidPrice,
		YesBidQty:         bestBidQty,
		YesAskPriceBps:    bestAskPrice,
		YesAskQty:         bestAskQty,
		ReceiveTs:         b.lastUpdate,
		Sequence:          b.lastSequence,
		Stale:             b.IsStale(),
	}, nil
}

func (b *Book) Snapshot(depth int) ([]domain.PriceLevel, []domain.PriceLevel, uint64, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	bids := sortedBids(b.yesBids, depth)
	asks := sortedAsks(b.yesAsks, depth)

	return bids, asks, b.lastSequence, b.IsStale()
}

func (b *Book) IsStale() bool {
	if b.lastUpdate.IsZero() {
		return true
	}

	return time.Since(b.lastUpdate) > b.staleAfter
}

func bestBid(levels map[domain.PriceBps]int64) (domain.PriceBps, int64, bool) {
	var bestPrice domain.PriceBps
	var bestQty int64
	found := false

	for price, qty := range levels {
		if qty <= 0 {
			continue
		}

		if !found || price > bestPrice {
			bestPrice = price
			bestQty = qty
			found = true
		}
	}

	return bestPrice, bestQty, found
}

func bestAsk(levels map[domain.PriceBps]int64) (domain.PriceBps, int64, bool) {
	var bestPrice domain.PriceBps
	var bestQty int64
	found := false

	for price, qty := range levels {
		if qty <= 0 {
			continue
		}

		if !found || price < bestPrice {
			bestPrice = price
			bestQty = qty
			found = true
		}
	}

	return bestPrice, bestQty, found
}

func sortedBids(levels map[domain.PriceBps]int64, depth int) []domain.PriceLevel {
	out := make([]domain.PriceLevel, 0, len(levels))

	for price, qty := range levels {
		if qty > 0 {
			out = append(out, domain.PriceLevel{PriceBps: price, Quantity: qty})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].PriceBps > out[j].PriceBps
	})

	if depth > 0 && len(out) > depth {
		return out[:depth]
	}

	return out
}

func sortedAsks(levels map[domain.PriceBps]int64, depth int) []domain.PriceLevel {
	out := make([]domain.PriceLevel, 0, len(levels))

	for price, qty := range levels {
		if qty > 0 {
			out = append(out, domain.PriceLevel{PriceBps: price, Quantity: qty})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].PriceBps < out[j].PriceBps
	})

	if depth > 0 && len(out) > depth {
		return out[:depth]
	}

	return out
}
