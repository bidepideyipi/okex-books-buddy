## okex-buddy

### Local Development

### Prerequisites

- Go 1.20+
- Redis 6.x (running on localhost:6379)
- Python 3.9+ (for Bytewax, later milestones)
- Node.js 16+ (for Vue frontend, later milestones)

### M2 - WebSocket Client Setup

#### Proxy Configuration

The WebSocket client supports SOCKS5 proxy for local development:

- **Development (local)**: Enable proxy in `config/app.dev.env`
  ```bash
  USE_PROXY=true
  PROXY_ADDR=127.0.0.1:4781
  ```

- **Production (Hong Kong server)**: Disable proxy in `config/app.prod.env`
  ```bash
  USE_PROXY=false
  PROXY_ADDR=
  ```

The client will automatically use SOCKS5 proxy when `USE_PROXY=true` is set.

#### Setup Steps

1. **Start Redis** (required for M2)
   ```bash
   redis-server
   ```

2. **Configure trading pairs in Redis**
   ```bash
   # Add trading pairs to monitor (use SWAP contracts for real-time data)
   redis-cli SADD trading_pairs:active BTC-USDT-SWAP ETH-USDT-SWAP
   
   # Verify configuration
   redis-cli SMEMBERS trading_pairs:active
   ```

3. **Start WebSocket Client**
   ```bash
   cd /Users/anthony/Documents/github/okex-buddy
   
   # Load environment variables
   export $(grep -v '^#' config/app.dev.env | xargs)
   export $(grep -v '^#' config/influxdb.dev.env | xargs)
   
   # Run WebSocket client
   cd backend/go
   go run ./cmd/websocket_client
   ```

   **Expected output:**
   ```
   2026/01/01 10:10:55 Subscription confirmed: BTC-USDT-SWAP
   2026/01/01 10:10:55 Subscription confirmed: ETH-USDT-SWAP
   ```

4. **Verify subscription success**
   
   After the WebSocket client starts and subscriptions are confirmed, verify that order book data is being received:
   
   ```bash
   # Check order book snapshot (first 25 lines show metadata and first few price levels)
   redis-cli -h localhost -p 6379 HGETALL orderbook:BTC-USDT-SWAP | head -25
   
   # Expected output includes:
   # - instrument_id: BTC-USDT-SWAP
   # - checksum: <int32 value>
   # - asks: [array of ask price levels]
   # - bids: [array of bid price levels]
   # - timestamp: <unix timestamp>
   
   # Check real-time event stream length
   redis-cli LLEN list:orderbook:events
   # Should show increasing numbers as updates arrive
   ```
   
   If you see order book data with valid checksums and increasing event counts, the subscription is working correctly!

5. **Monitor in real-time**
   ```bash
   # Watch order book updates in Redis
   redis-cli MONITOR
   
   # Check order book snapshot for a specific pair
   redis-cli HGETALL orderbook:BTC-USDT-SWAP
   
   # Check system monitoring
   redis-cli HGETALL system:monitoring
   
   # View event stream
   redis-cli LRANGE list:orderbook:events 0 10
   ```

6. **Test dynamic subscription**
   ```bash
   # Add a new trading pair (will be subscribed in ~20 seconds)
   redis-cli SADD trading_pairs:active SOL-USDT-SWAP
   
   # Remove a trading pair (will be unsubscribed in ~20 seconds)
   redis-cli SREM trading_pairs:active ETH-USDT-SWAP
   ```

### Original Dev Setup (M1)

For reference, the original M1 setup instructions: (dev)

#### 1. Prerequisites
- **Golang**: 1.20+
- **Python**: 3.9+ (for Bytewax / analysis flow)
- **Node.js**: 16+ (for Vue frontend)
- **Redis**: 6.x (running locally, e.g. `localhost:6379`)
- **InfluxDB**: 2.x (already running on `http://localhost:8086` per your setup)

#### 2. Environment configuration

From the project root (`/Users/anthony/Documents/github/okex-buddy`), load dev environment variables:

```bash
cd /Users/anthony/Documents/github/okex-buddy

# App-level dev config (Redis, OKEx WS, API bind, etc.)
export $(grep -v '^#' config/app.dev.env | xargs)

# InfluxDB dev config (only if you need Influx in the running service)
export $(grep -v '^#' config/influxdb.dev.env | xargs)
```

> Keep real secrets (passwords, tokens) in local `.env` or your shell, do not commit them.

#### 3. Start backend services

- **WebSocket client service (Go)**

```bash
cd /Users/anthony/Documents/github/okex-buddy/backend/go
go run ./cmd/websocket_client
```

- **API / monitoring backend service (Go)**

```bash
cd /Users/anthony/Documents/github/okex-buddy/backend/go
go run ./cmd/api_server
```

Both services automatically load configuration from environment variables via the unified `internal/config` module. Make sure you've exported the env vars (step 2) before running.

#### 4. Start analysis flow (Python / Bytewax)

Current skeleton (before wiring Bytewax `Dataflow`) can be run with:

```bash
cd /Users/anthony/Documents/github/okex-buddy
export $(grep -v '^#' config/influxdb.dev.env | xargs)
python analysis/bytewax/analysis_flow.py
```

Once the Bytewax flow is implemented, you can also use:

```bash
cd /Users/anthony/Documents/github/okex-buddy/analysis/bytewax
bytewax run analysis_flow.py
```

#### 5. Start frontend (Vue monitoring app)

The Vue 3 dashboard is now fully implemented with real-time analysis visualization:

```bash
cd /Users/anthony/Documents/github/okex-buddy/frontend/monitoring
npm install
npm run dev
```

The dashboard will be available at `http://localhost:5173` and automatically proxy API requests to the Go API server at `localhost:8080`.

**Features:**
- Real-time support/resistance level display
- Large order distribution analysis with sentiment indicators
- Interactive ECharts pie chart visualization
- Auto-refresh every 2 seconds
- Responsive design with Element Plus UI

See [frontend/monitoring/README.md](frontend/monitoring/README.md) for detailed documentation.

#### 6. Process overview in dev
- **Go WebSocket client**: connects to OKEx WS, processes order book data, writes into Redis / InfluxDB.
- **Python / Bytewax analysis**: consumes buffered data (via Redis List), computes deeper analytics, writes results to Redis / InfluxDB.
- **Go API server**: exposes REST/WS endpoints for current state and historical data.
- **Vue monitoring app**: connects to the API/WS endpoints to visualize metrics, order book, and alerts.
