package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/net/proxy"

	"github.com/supermancell/okex-buddy/internal/common"
)

// PublicClient manages the WebSocket connection to OKEx public channel
type PublicClient struct {
	url            string
	conn           *websocket.Conn
	mu             sync.RWMutex
	msgHandler     common.MessageHandler
	reconnectDelay time.Duration
	maxReconnect   int
	ctx            context.Context
	cancel         context.CancelFunc
	subscribed     map[string]bool // track subscribed instruments
	subscribedMu   sync.RWMutex
	useProxy       bool
	proxyAddr      string
	pingInterval   time.Duration
	pongTimeout    time.Duration
}

// NewPublicClient creates a new WebSocket client
func NewPublicClient(url string, msgHandler common.MessageHandler) *PublicClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &PublicClient{
		url:            url,
		msgHandler:     msgHandler,
		reconnectDelay: 5 * time.Second,
		maxReconnect:   3,
		ctx:            ctx,
		cancel:         cancel,
		subscribed:     make(map[string]bool),
		pingInterval:   25 * time.Second,
		pongTimeout:    30 * time.Second,
	}
}

// NewPublicClientWithProxy creates a new WebSocket client with proxy support
func NewPublicClientWithProxy(url string, msgHandler common.MessageHandler, useProxy bool, proxyAddr string) *PublicClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &PublicClient{
		url:            url,
		msgHandler:     msgHandler,
		reconnectDelay: 5 * time.Second,
		maxReconnect:   3,
		ctx:            ctx,
		cancel:         cancel,
		subscribed:     make(map[string]bool),
		useProxy:       useProxy,
		proxyAddr:      proxyAddr,
		pingInterval:   25 * time.Second,
		pongTimeout:    30 * time.Second,
	}
}

// Connect establishes the WebSocket connection
func (c *PublicClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	// Configure SOCKS5 proxy if enabled
	if c.useProxy && c.proxyAddr != "" {
		log.Printf("Using SOCKS5 proxy: %s", c.proxyAddr)
		dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Create SOCKS5 proxy dialer
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
	log.Printf("WebSocket connected to %s", c.url)

	// Start message reader in goroutine
	go c.readMessages()
	go c.startPingPong()

	return nil
}

// readMessages continuously reads messages from WebSocket
func (c *PublicClient) readMessages() {
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
				log.Printf("Error reading message: %v", err)
				// Trigger reconnection
				go c.reconnect()
				return
			}

			// Handle message
			if c.msgHandler != nil {
				if err := c.msgHandler(message); err != nil {
					log.Printf("Error handling message: %v", err)
				}
			}
		}
	}
}

// reconnect attempts to reconnect with exponential backoff
func (c *PublicClient) reconnect() {
	for attempt := 1; attempt <= c.maxReconnect; attempt++ {
		select {
		case <-c.ctx.Done():
			return
		default:
			delay := c.reconnectDelay * time.Duration(attempt)
			log.Printf("Reconnecting in %v (attempt %d/%d)", delay, attempt, c.maxReconnect)
			time.Sleep(delay)

			if err := c.Connect(); err != nil {
				log.Printf("Reconnect attempt %d failed: %v", attempt, err)
				continue
			}

			log.Println("Reconnected successfully")
			// Resubscribe to all instruments
			c.resubscribeAll()
			return
		}
	}

	log.Printf("Failed to reconnect after %d attempts", c.maxReconnect)
}

// Subscribe subscribes to both order book and ticker data for trading pairs
func (c *PublicClient) Subscribe(params interface{}) error {
	instruments, ok := params.([]string)
	if !ok {
		return fmt.Errorf("invalid params type for PublicClient Subscribe, expected []string")
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	// Build subscription message for both channels according to OKEx API
	args := make([]map[string]string, 0, len(instruments)*2)

	// Subscribe to both channels for each instrument
	for _, inst := range instruments {
		// Subscribe to books channel (order book data)
		args = append(args, map[string]string{
			"channel": "books",
			"instId":  inst,
		})

		// Subscribe to tickers channel (market data)
		args = append(args, map[string]string{
			"channel": "tickers",
			"instId":  inst,
		})
	}

	subMsg := map[string]interface{}{
		"op":   "subscribe",
		"args": args,
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

	// Track subscribed instruments
	c.subscribedMu.Lock()
	for _, inst := range instruments {
		c.subscribed[inst] = true
	}
	c.subscribedMu.Unlock()

	log.Printf("Subscribed to instruments: %v (both books and tickers channels)", instruments)
	return nil
}

// Unsubscribe unsubscribes from both order book and ticker data
func (c *PublicClient) Unsubscribe(params interface{}) error {
	instruments, ok := params.([]string)
	if !ok {
		return fmt.Errorf("invalid params type for PublicClient Unsubscribe, expected []string")
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	// Build unsubscription message for both channels
	args := make([]map[string]string, 0, len(instruments)*2)

	// Unsubscribe from both channels for each instrument
	for _, inst := range instruments {
		// Unsubscribe from books channel
		args = append(args, map[string]string{
			"channel": "books",
			"instId":  inst,
		})

		// Unsubscribe from tickers channel
		args = append(args, map[string]string{
			"channel": "tickers",
			"instId":  inst,
		})
	}

	unsubMsg := map[string]interface{}{
		"op":   "unsubscribe",
		"args": args,
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

	// Remove from subscribed tracking
	c.subscribedMu.Lock()
	for _, inst := range instruments {
		delete(c.subscribed, inst)
	}
	c.subscribedMu.Unlock()

	log.Printf("Unsubscribed from instruments: %v (both books and tickers channels)", instruments)
	return nil
}

// GetSubscribed returns the list of currently subscribed instruments
func (c *PublicClient) GetSubscribed() []string {
	c.subscribedMu.RLock()
	defer c.subscribedMu.RUnlock()

	instruments := make([]string, 0, len(c.subscribed))
	for inst := range c.subscribed {
		instruments = append(instruments, inst)
	}
	return instruments
}

// resubscribeAll resubscribes to all previously subscribed instruments
func (c *PublicClient) resubscribeAll() {
	instruments := c.GetSubscribed()
	if len(instruments) > 0 {
		log.Printf("Resubscribing to %d instruments", len(instruments))
		if err := c.Subscribe(instruments); err != nil {
			log.Printf("Failed to resubscribe: %v", err)
		}
	}
}

// startPingPong sends periodic ping messages to keep the connection alive
func (c *PublicClient) startPingPong() {
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
				log.Printf("Failed to send ping on WebSocket: %v", err)
				return
			}
			log.Printf("[DEBUG] Public WebSocket ping sent")
		}
	}
}

// Close gracefully closes the WebSocket connection
func (c *PublicClient) Close() error {
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
		return err
	}

	return nil
}
