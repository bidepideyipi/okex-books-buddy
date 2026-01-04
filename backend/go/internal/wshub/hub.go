package wshub

/*hub.go 是OKEx订单簿实时分析系统中连接前端和后端的关键组件，
实现了高效、可靠的WebSocket连接管理和消息通信机制，为实时数据展示提供了技术支持。
它确保了前端监控界面能够实时接收最新的订单簿分析结果，同时支持动态订阅和资源优化。
*/
import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message types sent to clients
const (
	MessageTypeAnalysisUpdate = "analysis_update"
	MessageTypePing           = "ping"
	MessageTypePong           = "pong"
	MessageTypeError          = "error"
	MessageTypeSubscribe      = "subscribe"
	MessageTypeUnsubscribe    = "unsubscribe"
)

// Message represents a WebSocket message
type Message struct {
	Type         string                 `json:"type"`
	InstrumentID string                 `json:"instrument_id,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Timestamp    int64                  `json:"timestamp"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	subscribed map[string]bool
	mu         sync.RWMutex
}

// Hub manages WebSocket client connections and broadcasts
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's main loop
/*
Run() 函数启动主循环：
├── 处理客户端注册/注销
├── 广播消息到所有客户端
├── 定期发送ping心跳
└── 管理连接生命周期
*/
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("WebSocket client disconnected (total: %d)", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client send buffer is full, disconnect
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()

		case <-ticker.C:
			// Send ping to all clients
			pingMsg := Message{
				Type:      MessageTypePing,
				Timestamp: time.Now().Unix(),
			}
			data, _ := json.Marshal(pingMsg)
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- data:
				default:
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastAnalysisUpdate sends analysis update to all subscribed clients
func (h *Hub) BroadcastAnalysisUpdate(instrumentID string, data map[string]interface{}) {
	msg := Message{
		Type:         MessageTypeAnalysisUpdate,
		InstrumentID: instrumentID,
		Data:         data,
		Timestamp:    time.Now().Unix(),
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal analysis update: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		client.mu.RLock()
		isSubscribed := client.subscribed[instrumentID]
		client.mu.RUnlock()

		if isSubscribed {
			select {
			case client.send <- jsonData:
			default:
				// Skip if send buffer is full
			}
		}
	}
}

// Subscribe adds instrument to client's subscription list
func (c *Client) Subscribe(instrumentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscribed[instrumentID] = true
	log.Printf("Client subscribed to %s", instrumentID)
}

// Unsubscribe removes instrument from client's subscription list
func (c *Client) Unsubscribe(instrumentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscribed, instrumentID)
	log.Printf("Client unsubscribed from %s", instrumentID)
}

// readPump reads messages from the WebSocket connection
/*
readPump() 函数负责从 WebSocket 连接读取消息：
├── 设置读取超时
├── 处理 pong 心跳响应
├── 解析 JSON 消息
├── 处理订阅/取消订阅请求
└── 处理其他消息类型
*/
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse client message: %v", err)
			continue
		}

		switch msg.Type {
		case MessageTypeSubscribe:
			if msg.InstrumentID != "" {
				c.Subscribe(msg.InstrumentID)
			}
		case MessageTypeUnsubscribe:
			if msg.InstrumentID != "" {
				c.Unsubscribe(msg.InstrumentID)
			}
		case MessageTypePong:
			// Heartbeat response
		}
	}
}

// writePump writes messages to the WebSocket connection
/*
writePump() 函数负责将消息发送到 WebSocket 连接：
├── 从send通道获取消息
├── 发送到WebSocket连接
├── 定期发送ping消息
└── 错误处理与连接关闭
*/
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs handles WebSocket upgrade and client management
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		hub:        h,
		conn:       conn,
		send:       make(chan []byte, 256),
		subscribed: make(map[string]bool),
	}

	h.register <- client

	// Start read and write pumps in separate goroutines
	go client.writePump()
	go client.readPump()
}
