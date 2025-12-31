## okex-buddy

### Local Development (dev)

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

> The `frontend/monitoring` directory is currently a placeholder. After scaffolding the Vue 3 app (e.g. via Vite), typical dev commands will look like:

```bash
cd /Users/anthony/Documents/github/okex-buddy/frontend/monitoring
npm install
npm run dev
```

The frontend will consume the APIs and WebSocket endpoints exposed by the Go API server, and display metrics and analysis results.

#### 6. Process overview in dev
- **Go WebSocket client**: connects to OKEx WS, processes order book data, writes into Redis / InfluxDB.
- **Python / Bytewax analysis**: consumes buffered data (via Redis List), computes deeper analytics, writes results to Redis / InfluxDB.
- **Go API server**: exposes REST/WS endpoints for current state and historical data.
- **Vue monitoring app**: connects to the API/WS endpoints to visualize metrics, order book, and alerts.
