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
  - [ ] Internal in-memory buffering for low-latency path
  - [ ] Support / resistance level calculation
  - [ ] Large order distribution analysis
  - [ ] Write analysis results into Redis Hash (`analysis:{instrument_id}`)
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
  - [ ] Create Vue 3 app skeleton (Vite-based)
  - [ ] Implement WebSocket connection & status indicators
  - [ ] Implement real-time charts: orderbook metrics, latency, alerts
  - [ ] Implement pair selection, panel customization, and basic alert views
  - [ ] Implement historical data views (InfluxDB-backed)
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
  - [ ] Implement simple price level clustering / aggregation
  - [ ] Define thresholds & configuration (per pair)
- **Large Order Distribution**
  - [ ] Identify “large” orders by configurable size
  - [ ] Aggregate by price zones and side (buy/sell)
- **Output**
  - [ ] Map computations to Redis Hash fields under `analysis:{instrument_id}`
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
  - [ ] `GET /api/analysis/{instrument_id}` (current state from Redis)
  - [ ] `GET /api/history/orderbook` (InfluxDB query)
  - [ ] `GET /api/history/analysis` (InfluxDB query)
- **WebSocket APIs**
  - [ ] `ws://.../metrics` for real-time system metrics and alerts
- **Health & Monitoring**
  - [ ] `/healthz` & `/readyz` endpoints
  - [ ] Basic rate limiting / protection (optional)

#### 3.6 Vue Monitoring Application

- **Foundation**
  - [ ] Routing, layout, theme, basic auth (if any)
  - [ ] API and WebSocket client modules
- **Dashboards**
  - [ ] WebSocket connection & system metrics dashboard
  - [ ] Per-pair orderbook / analysis dashboard
  - [ ] Alerts list and detail view
  - [ ] Historical charts (via InfluxDB APIs)
- **UX Functionalities**
  - [ ] Pair selection & favorite pairs
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

- **Foundations**: DONE (repo structure ✓, config ✓)
- **WebSocket ingestion & buffering**: DONE (M2 ✓)
- **Real-time analysis (Go)**: TODO
- **Bytewax stream processing**: TODO
- **Storage integration (Redis / InfluxDB)**: TODO
- **API & backend monitoring service**: TODO
- **Vue monitoring frontend**: TODO
- **Deployment & observability**: TODO

> This file should be updated as implementation progresses: mark checklist items and milestone sections from TODO → IN_PROGRESS → DONE, and refine tasks as the architecture evolves.
