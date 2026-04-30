APP_NAME=hip4-prediction-gateway

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test ./...

.PHONY: proto
proto:
	protoc --go_out=. --go-grpc_out=. proto/marketdata.proto proto/router.proto

.PHONY: run-hip4
run-hip4:
	go run ./cmd/hip4-node --config configs/hip4-testnet.yaml

.PHONY: run-polymarket
run-polymarket:
	go run ./cmd/polymarket-node --config configs/polymarket.yaml

.PHONY: run-kalshi
run-kalshi:
	go run ./cmd/kalshi-node --config configs/kalshi.yaml

.PHONY: run-aggregator
run-aggregator:
	go run ./cmd/aggregator --config configs/aggregator.yaml

.PHONY: run-router
run-router:
	go run ./cmd/router --config configs/router.yaml

.PHONY: cli-aggregator
cli-aggregator:
	go run ./cmd/cli --mode aggregator --addr localhost:50060 --market HIP4_TESTNET_BTC_OUTCOME

.PHONY: cli-buy
cli-buy:
	go run ./cmd/cli --mode route --addr localhost:50070 --market HIP4_TESTNET_BTC_OUTCOME --side BUY_YES --qty 100

.PHONY: cli-sell
cli-sell:
	go run ./cmd/cli --mode route --addr localhost:50070 --market HIP4_TESTNET_BTC_OUTCOME --side SELL_YES --qty 100

.PHONY: cli-opps
cli-opps:
	go run ./cmd/cli --mode opportunities --addr localhost:50070 --market HIP4_TESTNET_BTC_OUTCOME

.PHONY: docker-up
docker-up:
	docker compose up --build

.PHONY: docker-down
docker-down:
	docker compose down

.PHONY: docker-logs
docker-logs:
	docker compose logs -f
