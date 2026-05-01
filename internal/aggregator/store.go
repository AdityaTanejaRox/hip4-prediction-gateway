package aggregator

import (
	"sort"
	"sync"
	"time"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
	"github.com/AdityaTanejaRox/hip4-prediction-gateway/internal/domain"
)

type VenueTopOfBook struct {
	Venue             string
	VenueMarketID     string
	CanonicalMarketID string

	YesBidPriceBps domain.PriceBps
	YesBidQty      int64

	YesAskPriceBps domain.PriceBps
	YesAskQty      int64

	ReceiveTs time.Time
	Sequence  uint64
	Stale     bool
}

type Store struct {
	mu sync.RWMutex

	quoteStaleAfter time.Duration

	// canonical_market_id -> venue -> top of book
	books map[string]map[string]VenueTopOfBook

	subscribers map[chan *pb.ConsolidatedBook]struct{}
}

func NewStore(quoteStaleAfter time.Duration) *Store {
	if quoteStaleAfter <= 0 {
		quoteStaleAfter = 3 * time.Second
	}

	return &Store{
		quoteStaleAfter: quoteStaleAfter,
		books:           make(map[string]map[string]VenueTopOfBook),
		subscribers:     make(map[chan *pb.ConsolidatedBook]struct{}),
	}
}

func (s *Store) Upsert(update VenueTopOfBook) {
	s.mu.Lock()

	if _, ok := s.books[update.CanonicalMarketID]; !ok {
		s.books[update.CanonicalMarketID] = make(map[string]VenueTopOfBook)
	}

	s.books[update.CanonicalMarketID][update.Venue] = update

	book := s.buildConsolidatedBookLocked(update.CanonicalMarketID)

	for ch := range s.subscribers {
		select {
		case ch <- book:
		default:
		}
	}

	s.mu.Unlock()
}

func (s *Store) GetConsolidatedBook(canonicalMarketID string) *pb.ConsolidatedBook {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.buildConsolidatedBookLocked(canonicalMarketID)
}

func (s *Store) Subscribe() chan *pb.ConsolidatedBook {
	ch := make(chan *pb.ConsolidatedBook, 1024)

	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()

	return ch
}

func (s *Store) Unsubscribe(ch chan *pb.ConsolidatedBook) {
	s.mu.Lock()
	delete(s.subscribers, ch)
	close(ch)
	s.mu.Unlock()
}

func (s *Store) buildConsolidatedBookLocked(canonicalMarketID string) *pb.ConsolidatedBook {
	venueBooks := s.books[canonicalMarketID]

	bids := make([]*pb.VenueQuote, 0)
	asks := make([]*pb.VenueQuote, 0)

	now := time.Now()

	for _, tob := range venueBooks {
		isStale := tob.Stale || tob.ReceiveTs.IsZero() || now.Sub(tob.ReceiveTs) > s.quoteStaleAfter

		if tob.YesBidQty > 0 {
			bids = append(bids, &pb.VenueQuote{
				Venue:         tob.Venue,
				VenueMarketId: tob.VenueMarketID,
				PriceBps:      int64(tob.YesBidPriceBps),
				Quantity:      tob.YesBidQty,
				Sequence:      tob.Sequence,
				Stale:         isStale,
				ReceiveTsNs:   tob.ReceiveTs.UnixNano(),
			})
		}

		if tob.YesAskQty > 0 {
			asks = append(asks, &pb.VenueQuote{
				Venue:         tob.Venue,
				VenueMarketId: tob.VenueMarketID,
				PriceBps:      int64(tob.YesAskPriceBps),
				Quantity:      tob.YesAskQty,
				Sequence:      tob.Sequence,
				Stale:         isStale,
				ReceiveTsNs:   tob.ReceiveTs.UnixNano(),
			})
		}
	}

	sort.Slice(bids, func(i, j int) bool {
		return bids[i].PriceBps > bids[j].PriceBps
	})

	sort.Slice(asks, func(i, j int) bool {
		return asks[i].PriceBps < asks[j].PriceBps
	})

	return &pb.ConsolidatedBook{
		CanonicalMarketId: canonicalMarketID,
		YesBids:           bids,
		YesAsks:           asks,
		GeneratedTsNs:     now.UnixNano(),
	}
}
