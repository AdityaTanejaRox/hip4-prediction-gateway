package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	NodeID         string `yaml:"node_id"`
	Venue          string `yaml:"venue"`
	Environment    string `yaml:"environment"`
	GRPCListenAddr string `yaml:"grpc_listen_addr"`
	MockMode       bool   `yaml:"mock_mode"`

	Hyperliquid HyperliquidConfig `yaml:"hyperliquid"`
}

type HyperliquidConfig struct {
	WebSocketURL      string `yaml:"websocket_url"`
	InfoURL           string `yaml:"info_url"`
	Asset             string `yaml:"asset"`
	VenueMarketID     string `yaml:"venue_market_id"`
	CanonicalMarketID string `yaml:"canonical_market_id"`
	StaleAfterMS      int    `yaml:"stale_after_ms"`
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
