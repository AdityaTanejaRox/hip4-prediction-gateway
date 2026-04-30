package domain

import "time"

type Venue string

const (
	VenueHIP4       Venue = "HIP4"
	VenuePolyMarket Venue = "POLYMARKET"
	VenueKalshi     Venue = "KALSHI"
)

type MarketStatus string

const (
	MarketStatusUnknown MarketStatus = "UNKNOWN"
	MarketStatusOpen    MarketStatus = "OPEN"
	MarketStatusHalted  MarketStatus = "HALTED"
	MarketStatusSettled MarketStatus = "SETTLED"
)

type PriceBps int64

const (
	MinPriceBps PriceBps = 0
	MaxPriceBps PriceBps = 10000
)

type OutcomeSide string

const (
	BuyYes  OutcomeSide = "BUY_YES"
	SellYes OutcomeSide = "SELL_YES"
	BuyNo   OutcomeSide = "BUY_NO"
	SellNo  OutcomeSide = "SELL_NO"
)

type CanonicalMarket struct {
	CanonicalID     string
	Question        string
	YesLabel        string
	NoLabel         string
	ResolutionTime  time.Time
	ResolutionRules string
}

type VenueMarket struct {
	Venue             Venue
	VenueMarketID     string
	CanonicalMarketID string
	BaseAsset         string
	Status            MarketStatus
}

type PriceLevel struct {
	PriceBps PriceBps
	Quantity int64
}

type TopOfBook struct {
	Venue             Venue
	VenueMarketID     string
	CanonicalMarketID string

	YesBidPriceBps PriceBps
	YesBidQty      int64

	YesAskPriceBps PriceBps
	YesAskQty      int64

	ExchangeTs time.Time
	ReceiveTs  time.Time
	Sequence   uint64
	Stale      bool
}
