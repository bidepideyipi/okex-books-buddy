#!/bin/bash

# Quick start script for OKEx Buddy development environment
# This script starts all necessary services for the OKEx analysis system

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_ROOT"

echo "🚀 Starting OKEx Buddy Development Environment"
echo "================================================"

# Check prerequisites
command -v redis-cli >/dev/null 2>&1 || { echo "❌ Redis not found. Please install Redis first."; exit 1; }
command -v go >/dev/null 2>&1 || { echo "❌ Go not found. Please install Go 1.20+ first."; exit 1; }

# 1. Check if Redis is running
echo ""
echo "1️⃣  Checking Redis..."
if ! redis-cli ping >/dev/null 2>&1; then
    echo "⚠️  Redis is not running. Please start Redis first:"
    echo "   redis-server"
    exit 1
fi
echo "✅ Redis is running"

# 2. Configure trading pairs
echo ""
echo "2️⃣  Configuring trading pairs in Redis..."
redis-cli SADD trading_pairs:active BTC-USDT-SWAP ETH-USDT-SWAP SOL-USDT-SWAP >/dev/null
echo "✅ Configured pairs: BTC-USDT-SWAP, ETH-USDT-SWAP, SOL-USDT-SWAP"

# 3. Load environment variables
echo ""
echo "3️⃣  Loading environment configuration..."
export $(grep -v '^#' config/app.dev.env | xargs)
echo "✅ Environment loaded"

# 4. Start WebSocket Client in background
echo ""
echo "4️⃣  Starting WebSocket client..."
cd backend/go
go run ./cmd/websocket_client > /tmp/okex-websocket.log 2>&1 &
WS_PID=$!
cd "$PROJECT_ROOT"
echo "✅ WebSocket client started (PID: $WS_PID)"
echo "   Logs: tail -f /tmp/okex-websocket.log"

# Wait a bit for WebSocket to connect
sleep 3

# 5. Start API Server in background
echo ""
echo "5️⃣  Starting API server..."
cd backend/go
go run ./cmd/api_server > /tmp/okex-api.log 2>&1 &
API_PID=$!
cd "$PROJECT_ROOT"
echo "✅ API server started (PID: $API_PID)"
echo "   Logs: tail -f /tmp/okex-api.log"
echo "   API: http://localhost:8080"

# Wait for API to be ready
sleep 2

echo "================================================"
echo "🎉 All services started successfully!"
echo "================================================"
echo ""
echo "🔌 API Server:     http://localhost:8080"
echo "📝 WebSocket Logs: tail -f /tmp/okex-websocket.log"
echo "📝 API Logs:       tail -f /tmp/okex-api.log"
echo ""
echo "To stop all services:"
echo "  kill $WS_PID $API_PID"
echo ""
echo "Press Ctrl+C to stop"
echo "================================================"
echo ""

# Wait for processes to complete
wait $WS_PID $API_PID

# Cleanup on exit
trap "kill $WS_PID $API_PID 2>/dev/null" EXIT
