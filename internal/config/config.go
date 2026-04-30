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
	Polymarket  PolymarketConfig  `yaml:"polymarket"`
	Kalshi      KalshiConfig      `yaml:"kalshi"`
}

type HyperliquidConfig struct {
	WebSocketURL      string `yaml:"websocket_url"`
	InfoURL           string `yaml:"info_url"`
	Asset             string `yaml:"asset"`
	VenueMarketID     string `yaml:"venue_market_id"`
	CanonicalMarketID string `yaml:"canonical_market_id"`
	StaleAfterMS      int    `yaml:"stale_after_ms"`
}

type PolymarketConfig struct {
	WebSocketURL      string `yaml:"websocket_url"`
	VenueMarketID     string `yaml:"venue_market_id"`
	CanonicalMarketID string `yaml:"canonical_market_id"`
	StaleAfterMS      int    `yaml:"stale_after_ms"`
}

type KalshiConfig struct {
	WebSocketURL      string `yaml:"websocket_url"`
	VenueMarketID     string `yaml:"venue_market_id"`
	CanonicalMarketID string `yaml:"canonical_market_id"`
	StaleAfterMS      int    `yaml:"stale_after_ms"`
}

type AggregatorConfig struct {
	NodeID         string            `yaml:"node_id"`
	GRPCListenAddr string            `yaml:"grpc_listen_addr"`
	Markets        []string          `yaml:"markets"`
	VenueNodes     []VenueNodeConfig `yaml:"venue_nodes"`
}

type VenueNodeConfig struct {
	Name    string `yaml:"name"`
	Venue   string `yaml:"venue"`
	Address string `yaml:"address"`
}

type RouterConfig struct {
	NodeID         string `yaml:"node_id"`
	GRPCListenAddr string `yaml:"grpc_listen_addr"`

	Aggregator AggregatorClientConfig `yaml:"aggregator"`
	Markets    []string               `yaml:"markets"`
	Routing    RoutingConfig          `yaml:"routing"`
}

type AggregatorClientConfig struct {
	Address string `yaml:"address"`
}

type RoutingConfig struct {
	MinNetEdgeBps      int64 `yaml:"min_net_edge_bps"`
	DefaultFeeBps      int64 `yaml:"default_fee_bps"`
	DefaultSlippageBps int64 `yaml:"default_slippage_bps"`
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

func LoadAggregator(path string) (AggregatorConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return AggregatorConfig{}, err
	}

	var cfg AggregatorConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return AggregatorConfig{}, err
	}

	return cfg, nil
}

func LoadRouter(path string) (RouterConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return RouterConfig{}, err
	}

	var cfg RouterConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return RouterConfig{}, err
	}

	return cfg, nil
}
