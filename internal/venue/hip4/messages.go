package hip4

type subscriptionRequest struct {
	Method       string       `json:"method"`
	Subscription subscription `json:"subscription"`
}

type subscription struct {
	Type string `json:"type"`
	Coin string `json:"coin,omitempty"`
}

type hyperliquidEnveloper struct {
	Channel string `json:"channel"`
	Data    string `json:"data"`
}

type hyperliquidData struct {
	Coin   string          `json:"coin"`
	Time   int64           `json:"time"`
	Levels [][]l2BookLevel `json:"levels"`
}

type l2BookLevel struct {
	Px string `json:"px"`
	Sz string `json:"sz"`
	N  int64  `json:"n"`
}
