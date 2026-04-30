$ErrorActionPreference = "Stop"

Write-Host "Starting HIP4 node..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "go run ./cmd/hip4-node --config configs/hip4-testnet.yaml"

Start-Sleep -Seconds 2

Write-Host "Starting Polymarket node..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "go run ./cmd/polymarket-node --config configs/polymarket.yaml"

Start-Sleep -Seconds 2

Write-Host "Starting Aggregator..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "go run ./cmd/aggregator --config configs/aggregator.yaml"

Start-Sleep -Seconds 2

Write-Host "Starting Router..."
Start-Process powershell -ArgumentList "-NoExit", "-Command", "go run ./cmd/router --config configs/router.yaml"

Write-Host "All services launched."
Write-Host "HIP4:        localhost:50051"
Write-Host "Polymarket:  localhost:50052"
Write-Host "Aggregator:  localhost:50060"
Write-Host "Router:      localhost:50070"
