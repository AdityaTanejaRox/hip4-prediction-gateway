package polymarket

type wsSubscriptionRequest struct {
	AssetsIDs []string `json:"assets_ids"`
	Type      string   `json:"type"`
}

type wsEventEnvelope struct {
	EventType string `json:"event_type"`
	AssetID   string `json:"asset_id"`
	Market    string `json:"market"`
	Timestamp string `json:"timestamp"`
}

type bookEvent struct {
	EventType string       `json:"event_type"`
	AssetID   string       `json:"asset_id"`
	Market    string       `json:"market"`
	Timestamp string       `json:"timestamp"`
	Bids      []priceLevel `json:"bids"`
	Asks      []priceLevel `json:"asks"`
}

type priceChangeEvent struct {
	EventType string        `json:"event_type"`
	AssetID   string        `json:"asset_id"`
	Market    string        `json:"market"`
	Timestamp string        `json:"timestamp"`
	Changes   []priceChange `json:"changes"`
}

type priceChange struct {
	Side  string `json:"side"`
	Price string `json:"price"`
	Size  string `json:"size"`
}

type priceLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}
