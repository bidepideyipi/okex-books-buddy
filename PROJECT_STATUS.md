## Project Status & Development Roadmap

### 1. Scope & Targets

- **System**: OKEx order book real-time analysis & monitoring
- **Tech stack**: Go (WebSocket + core processing), Python + Bytewax (stream analysis), Redis
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
  - [ ] Persist selected analysis results / metrics into Redis Hash
- **M5 – Storage Layer (Redis)**
  - [ ] Implement Redis key conventions & TTL / persistence policies
- **M6 – API Layer & Monitoring Backend**
  - [ ] Design REST API (read from Redis)
  - [ ] Implement real-time metrics WebSocket for monitoring
  - [ ] Implement basic auth / API gateway integration (if required)
- **M7 – Ops, HA & Observability**
  - [ ] Containerize all services (Go, Bytewax, API)
  - [ ] Define docker-compose / K8s manifests for local & prod-like deploy
  - [ ] Add metrics / logging for Go, Bytewax, Redis
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
  - [x] Identify orders above percentile threshold
  - [x] Apply distance-based weighting (exponential decay from mid price)
  - [x] Calculate BullPower (weighted large buy orders) and BearPower (weighted large sell orders)
  - [x] Calculate sentiment indicator: (BullPower - BearPower) / (BullPower + BearPower)
  - [x] Store results in Redis Hash (`analysis:large_orders:{instrument_id}`) with sentiment field
  - [x] Expose via HTTP API with sentiment field
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
  - [ ] Write relevant metrics into Redis Hash (`analysis:depth_anomaly:{instrument_id}`, `analysis:liquidity_shrink:{instrument_id}`)

#### 3.4 Storage Design & Implementation

- **Redis**
  - [ ] Implement `orderbook:{instrument_id}` Hash
  - [ ] Implement `analysis:{instrument_id}` Hash

#### 3.5 API & Monitoring Backend

- **REST / HTTP APIs**
  - [x] `GET /api/analysis/{instrument_id}` (current state from Redis)
- **WebSocket APIs**
  - [ ] `ws://.../metrics` for real-time system metrics and alerts
- **Health & Monitoring**
  - [ ] `/healthz` & `/readyz` endpoints
  - [ ] Basic rate limiting / protection (optional)



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
- **Storage integration (Redis)**: ✅ COMPLETE
  - Redis: ✅ Analysis results storage complete
- **API & backend monitoring service**: ✅ DONE (M6 Basic)
  - Basic analysis API complete
- **Deployment & observability**: ⏳ TODO (M7)

### 6. Next Steps (Recommended Priority)

**Option A: Continue M3/M6 - Complete API Layer**
- Add health check endpoints (`/healthz`, `/readyz`)
- Add `/api/pairs` endpoint to list active trading pairs
- Add `/api/orderbook/{instrument_id}` to read latest order book snapshot

**Option B: Start M4 - Bytewax Stream Processing**
- Implement depth anomaly detection (Z-score based)
- Implement liquidity shrinkage warning (multi-window regression)
- Set up Bytewax dataflow to consume from Redis List
- Write results to `analysis:depth_anomaly:{instrument_id}` and `analysis:liquidity_shrink:{instrument_id}`

**Recommendation**: 
- **Option A** (Complete API layer basics) - This provides a stable foundation for monitoring and makes testing easier.
- Parallel track: **Option B** (Bytewax) can be developed independently as it uses separate Redis List input.

> This file should be updated as implementation progresses: mark checklist items and milestone sections from TODO → IN_PROGRESS → DONE, and refine tasks as the architecture evolves.
