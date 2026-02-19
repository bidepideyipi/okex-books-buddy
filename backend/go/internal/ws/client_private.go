package ws

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"

	"github.com/supermancell/okex-buddy/internal/common"
)

// OKExConfig holds OKEx API credentials
type OKExConfig struct {
	APIKey     string
	SecretKey  string
	Passphrase string
}

// timeOffset stores the time offset from server
var timeOffset int64 = 0

// PrivateClient manages the WebSocket connection to OKEx private channel
type PrivateClient struct {
	url            string
	conn           *websocket.Conn
	mu             sync.RWMutex
	msgHandler     common.MessageHandler
	reconnectDelay time.Duration
	maxReconnect   int
	ctx            context.Context
	cancel         context.CancelFunc
	subscribed     map[string]bool
	subscribedMu   sync.RWMutex
	useProxy       bool
	proxyAddr      string
	httpProxyAddr  string
	pingInterval   time.Duration
	pongTimeout    time.Duration
	config         OKExConfig
	authenticated  bool
	loginSuccess   chan bool
}

// NewPrivateClient creates a new private WebSocket client
func NewPrivateClient(url string, msgHandler common.MessageHandler, config OKExConfig) *PrivateClient {
	return NewPrivateClientWithProxy(url, msgHandler, false, "", config)
}

// NewPrivateClientWithProxy creates a new private WebSocket client with proxy support
func NewPrivateClientWithProxy(url string, msgHandler common.MessageHandler, useProxy bool, proxyAddr string, config OKExConfig) *PrivateClient {
	return NewPrivateClientWithDualProxy(url, msgHandler, useProxy, proxyAddr, "", config)
}

// NewPrivateClientWithDualProxy creates a new private WebSocket client with both SOCKS5 and HTTP proxy support
func NewPrivateClientWithDualProxy(url string, msgHandler common.MessageHandler, useProxy bool, proxyAddr string, httpProxyAddr string, config OKExConfig) *PrivateClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &PrivateClient{
		url:            url,
		msgHandler:     msgHandler,
		reconnectDelay: 5 * time.Second,
		maxReconnect:   3,
		ctx:            ctx,
		cancel:         cancel,
		subscribed:     make(map[string]bool),
		useProxy:       useProxy,
		proxyAddr:      proxyAddr,
		httpProxyAddr:  httpProxyAddr,
		pingInterval:   25 * time.Second,
		pongTimeout:    30 * time.Second,
		config:         config,
		authenticated:  false,
		loginSuccess:   make(chan bool, 1),
	}
}

// Connect establishes the WebSocket connection
func (c *PrivateClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	if c.useProxy && c.proxyAddr != "" {
		log.Printf("Using SOCKS5 proxy for private: %s", c.proxyAddr)
		dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			proxyDialer, err := proxy.SOCKS5("tcp", c.proxyAddr, nil, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("failed to create SOCKS5 proxy: %w", err)
			}
			return proxyDialer.Dial(network, addr)
		}
	}

	conn, _, err := dialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.url, err)
	}

	c.conn = conn
	c.authenticated = false
	log.Printf("Private WebSocket connected to %s", c.url)

	go c.readMessages()
	go c.startPingPong()

	return nil
}

// Login authenticates with OKEx using API credentials
func (c *PrivateClient) Login() error {
	log.Printf("Syncing time with OKEx server...")
	offset, err := SyncServerTime(c.httpProxyAddr)
	if err != nil {
		log.Printf("Warning: Failed to sync time: %v, using local time", err)
	} else {
		timeOffset = offset
	}

	timestamp := strconv.FormatInt((time.Now().UnixMilli()+timeOffset)/1000, 10)

	signStr := timestamp + "GET" + "/users/self/verify"
	h := hmac.New(sha256.New, []byte(c.config.SecretKey))
	h.Write([]byte(signStr))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	loginMsg := map[string]interface{}{
		"op": "login",
		"args": []map[string]string{
			{
				"apiKey":     c.config.APIKey,
				"passphrase": c.config.Passphrase,
				"timestamp":  timestamp,
				"sign":       signature,
			},
		},
	}

	log.Printf("Login timestamp: %s (local: %d, offset: %d ms)", timestamp, time.Now().UnixMilli(), timeOffset)

	data, err := json.Marshal(loginMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal login message: %w", err)
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return fmt.Errorf("failed to send login message: %w", err)
	}

	log.Printf("Private WebSocket login request sent")

	select {
	case success := <-c.loginSuccess:
		if success {
			c.mu.Lock()
			c.authenticated = true
			c.mu.Unlock()
			log.Println("Private WebSocket login successful")
			return nil
		}
		return fmt.Errorf("login failed")
	case <-time.After(10 * time.Second):
		return fmt.Errorf("login timeout")
	}
}

// readMessages continuously reads messages from WebSocket
func (c *PrivateClient) readMessages() {
	defer func() {
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading private message: %v", err)
				go c.reconnect()
				return
			}

			if c.handleMessage(message) {
				continue
			}

			if c.msgHandler != nil {
				if err := c.msgHandler(message); err != nil {
					log.Printf("Error handling private message: %v", err)
				}
			}
		}
	}
}

// handleMessage handles incoming WebSocket messages
// Returns true if message was handled internally and should not be passed to msgHandler
func (c *PrivateClient) handleMessage(message []byte) bool {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return false
	}

	event, ok := msg["event"].(string)
	if !ok {
		return false
	}

	if event == "login" {
		code, _ := msg["code"].(string)
		if code == "0" {
			c.loginSuccess <- true
		} else {
			log.Printf("Private WebSocket login failed: %v", msg)
			c.loginSuccess <- false
		}
		return true
	}

	if event == "error" {
		code, _ := msg["code"].(string)
		msgText, _ := msg["msg"].(string)
		log.Printf("Private WebSocket error: code=%s, msg=%s", code, msgText)

		if c.loginSuccess != nil {
			c.loginSuccess <- false
		}
		return true
	}

	return false
}

// reconnect attempts to reconnect with exponential backoff
func (c *PrivateClient) reconnect() {
	c.mu.Lock()
	c.authenticated = false
	c.mu.Unlock()

	for attempt := 1; attempt <= c.maxReconnect; attempt++ {
		select {
		case <-c.ctx.Done():
			return
		default:
			delay := c.reconnectDelay * time.Duration(attempt)
			log.Printf("Reconnecting private in %v (attempt %d/%d)", delay, attempt, c.maxReconnect)
			time.Sleep(delay)

			if err := c.Connect(); err != nil {
				log.Printf("Private reconnect attempt %d failed: %v", attempt, err)
				continue
			}

			if err := c.Login(); err != nil {
				log.Printf("Private login attempt %d failed: %v", attempt, err)
				continue
			}

			log.Println("Private reconnected and logged in successfully")
			c.resubscribeAll()
			return
		}
	}

	log.Printf("Failed to reconnect private after %d attempts", c.maxReconnect)
}

// Subscribe subscribes to private channels
func (c *PrivateClient) Subscribe(params interface{}) error {
	channels, ok := params.([]map[string]string)
	if !ok {
		return fmt.Errorf("invalid params type for PrivateClient Subscribe, expected []map[string]string")
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	subMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": channels,
	}

	data, err := json.Marshal(subMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal subscribe message: %w", err)
	}

	c.mu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	c.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	c.subscribedMu.Lock()
	for _, ch := range channels {
		key := fmt.Sprintf("%s:%s", ch["channel"], ch["instType"])
		c.subscribed[key] = true
	}
	c.subscribedMu.Unlock()

	log.Printf("Subscribed to private channels: %v", channels)
	return nil
}

// Unsubscribe unsubscribes from private channels
func (c *PrivateClient) Unsubscribe(params interface{}) error {
	channels, ok := params.([]map[string]string)
	if !ok {
		return fmt.Errorf("invalid params type for PrivateClient Unsubscribe, expected []map[string]string")
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	unsubMsg := map[string]interface{}{
		"op":   "unsubscribe",
		"args": channels,
	}

	data, err := json.Marshal(unsubMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscribe message: %w", err)
	}

	c.mu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	c.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send unsubscribe message: %w", err)
	}

	c.subscribedMu.Lock()
	for _, ch := range channels {
		key := fmt.Sprintf("%s:%s", ch["channel"], ch["instType"])
		delete(c.subscribed, key)
	}
	c.subscribedMu.Unlock()

	log.Printf("Unsubscribed from private channels: %v", channels)
	return nil
}

// PlaceOrder sends an order request via WebSocket
func (c *PrivateClient) PlaceOrder(args []map[string]string) error {
	c.mu.RLock()
	conn := c.conn
	authenticated := c.authenticated
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	if !authenticated {
		return fmt.Errorf("not authenticated")
	}

	orderMsg := map[string]interface{}{
		"id":   strconv.FormatInt(time.Now().UnixMilli(), 10),
		"op":   "order",
		"args": args,
	}

	data, err := json.Marshal(orderMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal order message: %w", err)
	}

	c.mu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	c.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to send order message: %w", err)
	}

	log.Printf("Order sent: %v", args)
	return nil
}

// GetSubscribed returns the list of currently subscribed channels
func (c *PrivateClient) GetSubscribed() []string {
	c.subscribedMu.RLock()
	defer c.subscribedMu.RUnlock()

	channels := make([]string, 0, len(c.subscribed))
	for ch := range c.subscribed {
		channels = append(channels, ch)
	}
	return channels
}

// resubscribeAll resubscribes to all previously subscribed channels
func (c *PrivateClient) resubscribeAll() {
	channels := c.GetSubscribed()
	if len(channels) > 0 {
		log.Printf("Resubscribing private to %d channels", len(channels))

		args := make([]map[string]string, 0)
		for _, ch := range channels {
			args = append(args, map[string]string{
				"channel":  ch,
				"instType": "SWAP",
			})
		}

		if err := c.Subscribe(args); err != nil {
			log.Printf("Failed to resubscribe private: %v", err)
		}
	}
}

// startPingPong sends periodic ping messages to keep the connection alive
func (c *PrivateClient) startPingPong() {
	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				return
			}

			err := conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Printf("Failed to send ping on private WebSocket: %v", err)
				return
			}
			log.Printf("[DEBUG] Private WebSocket ping sent")
		}
	}
}

// IsAuthenticated returns whether the client is authenticated
func (c *PrivateClient) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authenticated
}

// Close gracefully closes the WebSocket connection
func (c *PrivateClient) Close() error {
	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		if err != nil {
			log.Printf("Error sending close message: %v", err)
		}

		err = c.conn.Close()
		c.conn = nil
		c.authenticated = false
		return err
	}

	return nil
}

// ParseOrderID extracts ordId from order channel response
func ParseOrderID(message []byte) (string, error) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		return "", err
	}

	if event, ok := msg["event"].(string); ok && event == "order" {
		if data, ok := msg["data"].([]interface{}); ok && len(data) > 0 {
			if order, ok := data[0].(map[string]interface{}); ok {
				if ordId, ok := order["ordId"].(string); ok {
					return ordId, nil
				}
				if ordIdFloat, ok := order["ordId"].(float64); ok {
					return strconv.FormatFloat(ordIdFloat, 'f', -1, 64), nil
				}
			}
		}
	}

	return "", fmt.Errorf("order ID not found in message")
}
