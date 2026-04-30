#!/usr/bin/env bash
set -euo pipefail

go run ./cmd/cli \
  --mode route \
  --addr localhost:50070 \
  --market HIP4_TESTNET_BTC_OUTCOME \
  --side SELL_YES \
  --qty 100
