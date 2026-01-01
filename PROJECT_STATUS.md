## Project Status & Development Roadmap

### 1. Scope & Targets

- **System**: OKEx order book real-time analysis & monitoring
- **Tech stack**: Go (WebSocket + core processing), Python + Bytewax (stream analysis), Redis, InfluxDB, Vue 3 + Element Plus, ECharts
- **Key goals**:
  - **≤50 ms** end-to-end analysis latency
  - **Up to 10** trading pairs monitored concurrently
  - **High availability**, horizontal scalability, and observability

### 2. High-Level Milestones

- **M1 – Foundations & Environment**
  - [x] Initialize repo structure (backend Go services, Bytewax app, frontend Vue app)
  - [x] Define common configuration & environments (dev / test / prod)
  - [x] Unified config loading module in Go (`backend/go/internal/config`)
- **M2 – WebSocket Ingestion (Go)**
  - [x] Implement OKEx WebSocket client (multi-pair support, reconnect)
  - [x] Implement order book snapshot / incremental handling & checksum validation
  - [x] Implement data validation module
  - [x] Push normalized events into Redis List (message buffer)
  - [x] Store latest 400-depth snapshot per pair into Redis Hash
- **M3 – Real-Time Analysis (Go in-memory path)**
  - [x] Internal in-memory buffering for low-latency path
  - [x] Support / resistance level calculation
  - [x] Large order distribution analysis
  - [x] Write analysis results into Redis Hash (`analysis:support_resistance:{instrument_id}`, `analysis:large_orders:{instrument_id}`)
  - [x] Expose analysis data via HTTP API (`GET /api/analysis/{instrument_id}`)
- **M4 – Bytewax Stream Processing (Redis List path)**
  - [ ] Define Redis List message format for Bytewax
  - [ ] Implement Bytewax source for Redis List consumption
  - [ ] Implement depth anomaly detection (with time windows)
  - [ ] Implement liquidity shrinkage detection (longer windows)
  - [ ] Persist selected analysis results / metrics into InfluxDB
- **M5 – Storage Layer (InfluxDB & Redis)**
  - [ ] Stand up InfluxDB dev instance & schema (`orderbook_data`, `analysis_results`)
  - [ ] Implement write path from Go + Bytewax into InfluxDB
  - [ ] Implement Redis key conventions & TTL / persistence policies
- **M6 – API Layer & Monitoring Backend**
  - [ ] Design REST/WS API for Vue dashboard (read Redis / InfluxDB)
  - [ ] Implement historical query APIs (InfluxDB)
  - [ ] Implement real-time metrics WebSocket for frontend
  - [ ] Implement basic auth / API gateway integration (if required)
- **M7 – Frontend (Vue 3 + Element Plus + ECharts)**
  - [x] Create Vue 3 app skeleton (Vite-based)
  - [x] Implement API service module for fetching analysis data
  - [x] Implement real-time charts: support/resistance, large orders
  - [x] Implement pair selection and basic dashboard layout
  - [x] Configure CORS middleware in API server
  - [x] Implement WebSocket connection & status indicators
  - [x] Implement automatic fallback from WebSocket to polling
  - [ ] Implement historical data views (InfluxDB-backed)
  - [ ] Implement panel customization and alert views
- **M8 – Ops, HA & Observability**
  - [ ] Containerize all services (Go, Bytewax, API, Vue)
  - [ ] Define docker-compose / K8s manifests for local & prod-like deploy
  - [ ] Add metrics / logging for Go, Bytewax, Redis, InfluxDB
  - [ ] Set up alerting rules for latency, error rate, and resource usage

### 3. Component-Level Task Breakdown

#### 3.1 WebSocket Client & Data Preprocessing (Go)

- **Connection & Subscription**
  - [ ] Configurable list of trading pairs (max 10)
  - [ ] Connection lifecycle management (connect / reconnect / backoff)
  - [ ] Subscription / resubscription logic on reconnect
- **Message Handling**
  - [ ] Parse OKEx snapshot (`books`) messages
  - [ ] Parse incremental update messages (`action: update`)
  - [ ] Maintain in-memory order book per pair (400 levels)
  - [ ] Implement checksum verification & full resync on inconsistency
- **Validation & Buffering**
  - [ ] JSON schema / structural validation
  - [ ] Timestamps & field sanity checks
  - [ ] Push normalized events to Redis List for downstream processing
  - [ ] Update Redis Hash `orderbook:{instrument_id}` with latest snapshot

#### 3.2 Real-Time Analysis (Go in-memory path)

- **Support & Resistance**
  - [x] Implement simple price level clustering / aggregation
  - [x] Define thresholds & configuration (per pair)
- **Large Order Distribution**
  - [x] Identify "large" orders by configurable size
  - [x] Aggregate by price zones and side (buy/sell)
- **Output**
  - [x] Map computations to Redis Hash fields under `analysis:support_resistance:{instrument_id}` and `analysis:large_orders:{instrument_id}`
  - [ ] Expose metrics for processing latency

#### 3.3 Bytewax + Redis List Path (Python)

- **Integration**
  - [ ] Implement custom `Source` for Redis List (BRPOP-based)
  - [ ] Define serialization (JSON) & versioning for events
- **Depth Anomaly Detection**
  - [ ] Define windowing strategy (short time windows)
  - [ ] Implement depth change metrics & anomaly score
  - [ ] Set alert thresholds & mapping to `depth_anomaly_*` fields
- **Liquidity Shrinkage Detection**
  - [ ] Define longer time windows
  - [ ] Compute liquidity index & trends
  - [ ] Set alert thresholds & mapping to `liquidity_*` fields
- **Persistence**
  - [ ] Write relevant metrics into InfluxDB (`analysis_results`)

#### 3.4 Storage Design & Implementation

- **Redis**
  - [ ] Implement `orderbook:{instrument_id}` Hash
  - [ ] Implement `analysis:{instrument_id}` Hash
  - [ ] Implement `system:monitoring` Hash updates from backend services
- **InfluxDB**
  - [ ] Implement writes for `orderbook_data`
  - [ ] Implement writes for `analysis_results`
  - [ ] Implement retention policies and basic indexes / tags as per doc

#### 3.5 API & Monitoring Backend

- **REST / HTTP APIs**
  - [x] `GET /api/analysis/{instrument_id}` (current state from Redis)
  - [ ] `GET /api/history/orderbook` (InfluxDB query)
  - [ ] `GET /api/history/analysis` (InfluxDB query)
- **WebSocket APIs**
  - [ ] `ws://.../metrics` for real-time system metrics and alerts
- **Health & Monitoring**
  - [ ] `/healthz` & `/readyz` endpoints
  - [ ] Basic rate limiting / protection (optional)

#### 3.6 Vue Monitoring Application

- **Foundation**
  - [x] Routing, layout, theme, basic auth (if any)
  - [x] API and WebSocket client modules
- **Dashboards**
  - [ ] WebSocket connection & system metrics dashboard
  - [x] Per-pair orderbook / analysis dashboard
  - [ ] Alerts list and detail view
  - [ ] Historical charts (via InfluxDB APIs)
- **UX Functionalities**
  - [x] Pair selection & favorite pairs
  - [ ] Customizable panels & basic persistence (localStorage or backend)

### 4. Testing & Quality

- **Go services**
  - [ ] Unit tests for WebSocket client, parsers, analysis logic
  - [ ] Integration tests using mocked OKEx streams & Redis
- **Bytewax flows**
  - [ ] Unit tests for operators / functions
  - [ ] Integration tests with test Redis instance
- **Frontend**
  - [ ] Component tests for key charts / views
  - [ ] Basic e2e flows (smoke tests)

### 5. Current Overall Status

- **Foundations**: ✅ DONE (repo structure ✓, config ✓)
- **WebSocket ingestion & buffering**: ✅ DONE (M2 ✓)
- **Real-time analysis (Go)**: ✅ DONE (M3 ✓)
  - Support/resistance calculation implemented
  - Large order distribution analysis implemented
  - HTTP API endpoint for reading analysis results
- **Bytewax stream processing**: ⏳ TODO (M4)
- **Storage integration (Redis / InfluxDB)**: 🔄 PARTIAL
  - Redis: ✅ Analysis results storage complete
  - InfluxDB: ⏳ TODO (async write path)
- **API & backend monitoring service**: ✅ DONE (M6 Basic)
  - Basic analysis API complete
  - CORS middleware added
  - Historical queries TODO
- **Vue monitoring frontend**: ✅ DONE (M7 Complete)
  - ✅ Vue 3 + Vite + TypeScript setup
  - ✅ Element Plus + ECharts integration
  - ✅ Support/Resistance card component
  - ✅ Large Orders card with sentiment
  - ✅ ECharts pie chart visualization
  - ✅ WebSocket real-time push with automatic fallback
  - ✅ Connection status indicator with mode display
  - ✅ Auto-reconnection with exponential backoff
  - ⏳ Historical data views (future)
- **Deployment & observability**: ⏳ TODO (M8)

### 6. Next Steps (Recommended Priority)

**Option A: Continue M3/M6 - Complete API Layer**
- Add health check endpoints (`/healthz`, `/readyz`)
- Add CORS middleware for Vue frontend
- Add `/api/pairs` endpoint to list active trading pairs
- Add `/api/orderbook/{instrument_id}` to read latest order book snapshot

**Option B: Start M4 - Bytewax Stream Processing**
- Implement depth anomaly detection (Z-score based)
- Implement liquidity shrinkage warning (multi-window regression)
- Set up Bytewax dataflow to consume from Redis List
- Write results to `analysis:depth_anomaly:{instrument_id}` and `analysis:liquidity_shrink:{instrument_id}`

**Option C: Start M5 - InfluxDB Integration**
- Set up InfluxDB dev instance
- Implement async write worker in Go to persist analysis results to InfluxDB
- Define retention policies and schemas
- Add historical query API endpoints

**Option D: Start M7 - Vue Dashboard (MVP)**
- Bootstrap Vue 3 + Vite project
- Create basic layout with Element Plus
- Implement pair selector
- Display support/resistance and large order data from API
- Add simple ECharts visualization

**Recommendation**: 
- **Option A** (Complete API layer basics) - This provides a stable foundation for the Vue dashboard and makes testing easier.
- Then proceed to **Option D** (Vue MVP) to get end-to-end visualization working.
- Parallel track: **Option B** (Bytewax) can be developed independently as it uses separate Redis List input.

> This file should be updated as implementation progresses: mark checklist items and milestone sections from TODO → IN_PROGRESS → DONE, and refine tasks as the architecture evolves.
