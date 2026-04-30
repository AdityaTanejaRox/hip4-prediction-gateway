package router

import (
	"sync"

	pb "github.com/AdityaTanejaRox/hip4-prediction-gateway/generated/kairosnode"
)

type BookStore struct {
	mu sync.RWMutex

	books map[string]*pb.ConsolidatedBook
}

func NewBookStore() *BookStore {
	return &BookStore{
		books: make(map[string]*pb.ConsolidatedBook),
	}
}

func (s *BookStore) Upsert(book *pb.ConsolidatedBook) {
	s.mu.Lock()
	defer s.mu.Unlock()

	copied := cloneConsolidatedBook(book)
	s.books[book.CanonicalMarketId] = copied
}

func (s *BookStore) Get(canonicalMarketID string) (*pb.ConsolidatedBook, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	book, ok := s.books[canonicalMarketID]
	if !ok {
		return nil, false
	}

	return cloneConsolidatedBook(book), true
}

func cloneConsolidatedBook(book *pb.ConsolidatedBook) *pb.ConsolidatedBook {
	if book == nil {
		return nil
	}

	out := &pb.ConsolidatedBook{
		CanonicalMarketId: book.CanonicalMarketId,
		GeneratedTsNs:     book.GeneratedTsNs,
		YesBids:           make([]*pb.VenueQuote, 0, len(book.YesBids)),
		YesAsks:           make([]*pb.VenueQuote, 0, len(book.YesAsks)),
	}

	for _, bid := range book.YesBids {
		copied := *bid
		out.YesBids = append(out.YesBids, &copied)
	}

	for _, ask := range book.YesAsks {
		copied := *ask
		out.YesAsks = append(out.YesAsks, &copied)
	}

	return out
}
