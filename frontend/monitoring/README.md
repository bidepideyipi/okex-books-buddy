# OKEx Order Book Monitoring Dashboard

A real-time Vue 3 dashboard for monitoring OKEx order book analysis.

## Features

- 📊 **Real-time Analysis**: Support/Resistance levels and Large Order distribution
- ⚡ **WebSocket Updates**: Real-time push updates with automatic fallback to polling
- 🔄 **Auto-Reconnection**: Intelligent reconnection with exponential backoff
- 📡 **Connection Status**: Live status indicator showing WebSocket/Polling mode
- 🔍 **WebSocket Monitor**: Detailed connection monitoring page with real-time status
- 📈 **Interactive Charts**: ECharts-powered visualizations
- 🎨 **Modern UI**: Element Plus component library
- 📱 **Responsive**: Works on desktop and mobile devices

## Tech Stack

- **Vue 3** with Composition API and TypeScript
- **Vite** for fast development and builds
- **Element Plus** for UI components
- **ECharts** for data visualization
- **TypeScript** for type safety

## Development

### Prerequisites

- Node.js 16+ 
- Backend API server running on port 8080

### Setup

```bash
# Install dependencies
npm install

# Start development server (with API proxy)
npm run dev
```

The app will be available at `http://localhost:5173`

### Environment Variables

Create `.env.development` or `.env.production`:

```env
VITE_API_BASE_URL=http://localhost:8080
```

Note: In development, the Vite dev server proxies `/api/*` requests to `localhost:8080`, so you can leave this empty.

## Build

```bash
# Build for production
npm run build

# Preview production build
npm run preview
```

## Project Structure

```
src/
├── components/          # Vue components
│   ├── Layout.vue              # Main layout with sidebar
│   ├── ConnectionStatus.vue    # WebSocket connection status indicator
│   ├── WebSocketMonitor.vue    # WebSocket connection monitoring page
│   ├── SupportResistanceCard.vue
│   ├── LargeOrdersCard.vue
│   └── OrderDistributionChart.vue
├── services/            # API services
│   ├── api.ts                  # Backend REST API client
│   └── websocket.ts            # WebSocket client service
├── types/               # TypeScript types
│   └── analysis.ts
├── App.vue              # Root component
└── main.ts              # Application entry point
```

## API Integration

The dashboard connects to the Go backend via both REST API and WebSocket:

### REST API

#### Analysis Data
- **GET** `/api/analysis/{instrument_id}` - Fetch analysis data for a trading pair

#### WebSocket Connection Status
- **GET** `/api/websocket/status` - Get all WebSocket connection statuses
- **GET** `/api/websocket/status/{instrument_id}` - Get status for specific trading pair

### WebSocket
- **WS** `/ws/analysis` - Real-time analysis updates

**Message Protocol:**
```typescript
// Client to Server
{
  "type": "subscribe" | "unsubscribe",
  "instrument_id": "BTC-USDT",
  "timestamp": 1234567890
}

// Server to Client
{
  "type": "analysis_update",
  "instrument_id": "BTC-USDT",
  "data": {
    "support_resistance": { ... },
    "large_orders": { ... }
  },
  "timestamp": 1234567890
}
```

### REST Response Example:

**Analysis Data (`/api/analysis/{instrument_id}`):**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "instrument_id": "BTC-USDT",
    "support_resistance": {
      "instrument_id": "BTC-USDT",
      "analysis_time": "1735689600",
      "support_level_1": "42000.50",
      "support_level_2": "41500.25",
      "resistance_level_1": "43000.75",
      "resistance_level_2": "43500.00"
    },
    "large_orders": {
      "instrument_id": "BTC-USDT",
      "analysis_time": "1735689600",
      "large_buy_orders": "1250000.00",
      "large_sell_orders": "980000.00",
      "large_order_trend": "bullish"
    }
  }
}
```

**WebSocket Status (`/api/websocket/status`):**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total_connections": 3,
    "active_pairs": 3,
    "connections": [
      {
        "instrument_id": "BTC-USDT",
        "websocket_url": "wss://ws.okx.com:8443/ws/v5/public",
        "status": "connected",
        "connected_at": "2026-01-01T10:00:00Z",
        "last_message_at": "2026-01-01T10:05:30Z",
        "reconnect_count": 0,
        "messages_received": 1523,
        "last_error": ""
      }
    ]
  }
}
```

## Dashboard Views

The application has two main views:

### 1. Analysis Dashboard
- Real-time support/resistance levels
- Large order distribution analysis
- Interactive order distribution chart
- Auto-updates via WebSocket or polling

### 2. WebSocket Monitor
- Live connection status for all trading pairs
- Connection details (URL, connected time, last message time)
- Reconnection count tracking
- Messages received counter
- Auto-refresh every 5 seconds
- Detailed view for individual connections

## Connection Modes

The dashboard automatically manages two connection modes:

1. **WebSocket Mode (Primary)**
   - Real-time push updates (1s interval from server)
   - Lower latency and bandwidth usage
   - Automatic subscription management
   - Ping/pong heartbeat (30s)

2. **Polling Mode (Fallback)**
   - HTTP polling (2s interval)
   - Activates when WebSocket unavailable
   - Automatic switch when WebSocket reconnects

The connection status indicator shows current mode and allows manual reconnection.

## Available Trading Pairs

Currently configured pairs (can be extended):
- BTC-USDT
- ETH-USDT
- SOL-USDT
- BNB-USDT
- XRP-USDT

## Next Steps

- [x] WebSocket support for real-time push updates ✅
- [x] Connection status indicator with mode display ✅
- [x] Auto-reconnection with exponential backoff ✅
- [x] WebSocket connection monitoring page ✅
- [ ] Implement depth anomaly and liquidity shrinkage displays
- [ ] Add historical data charts from InfluxDB
- [ ] Add alert notifications
- [ ] Add user preferences and favorites
