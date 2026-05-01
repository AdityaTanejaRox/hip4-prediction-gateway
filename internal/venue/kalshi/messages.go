package kalshi

type wsSubscribeRequest struct {
	ID     int64          `json:"id"`
	Cmd    string         `json:"cmd"`
	Params map[string]any `json:"params"`
}

type wsEnvelope struct {
	Type string         `json:"type"`
	Sid  int64          `json:"sid,omitempty"`
	Seq  uint64         `json:"seq,omitempty"`
	Msg  jsonRawMessage `json:"msg,omitempty"`
}

type jsonRawMessage []byte

func (m *jsonRawMessage) UnmarshalJSON(data []byte) error {
	*m = append((*m)[0:0], data...)
	return nil
}

type orderbookSnapshotMessage struct {
	MarketTicker string    `json:"market_ticker"`
	Yes          [][]int64 `json:"yes"`
	No           [][]int64 `json:"no"`
}

type orderbookDeltaMessage struct {
	MarketTicker string `json:"market_ticker"`
	Price        int64  `json:"price"`
	Delta        int64  `json:"delta"`
	Side         string `json:"side"`
}
