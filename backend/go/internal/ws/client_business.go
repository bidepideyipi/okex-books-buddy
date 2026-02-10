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
)

// BusinessClient manages the WebSocket connection to OKEx business channel
type BusinessClient struct {
	url            string
	conn           *websocket.Conn
	mu             sync.RWMutex
	msgHandler     MessageHandler
	reconnectDelay time.Duration
	maxReconnect   int
	ctx            context.Context
	cancel         context.CancelFunc
	subscribed     map[string]bool
	subscribedMu   sync.RWMutex
	useProxy       bool
	proxyAddr      string
}

// NewBusinessClient creates a new business WebSocket client
func NewBusinessClient(url string, handler MessageHandler) *BusinessClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &BusinessClient{
		url:            url,
		msgHandler:     handler,
		reconnectDelay: 5 * time.Second,
		maxReconnect:   3,
		ctx:            ctx,
		cancel:         cancel,
		subscribed:     make(map[string]bool),
	}
}

// NewBusinessClientWithProxy creates a new business WebSocket client with proxy support
func NewBusinessClientWithProxy(url string, handler MessageHandler, useProxy bool, proxyAddr string) *BusinessClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &BusinessClient{
		url:            url,
		msgHandler:     handler,
		reconnectDelay: 5 * time.Second,
		maxReconnect:   3,
		ctx:            ctx,
		cancel:         cancel,
		subscribed:     make(map[string]bool),
		useProxy:       useProxy,
		proxyAddr:      proxyAddr,
	}
}

// Connect establishes the WebSocket connection
func (c *BusinessClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	if c.useProxy && c.proxyAddr != "" {
		log.Printf("Using SOCKS5 proxy for business: %s", c.proxyAddr)
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
	log.Printf("Business WebSocket connected to %s", c.url)

	go c.readMessages()

	return nil
}

// readMessages continuously reads messages from WebSocket
func (c *BusinessClient) readMessages() {
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
				log.Printf("Error reading business message: %v", err)
				go c.reconnect()
				return
			}

			if c.msgHandler != nil {
				if err := c.msgHandler(message); err != nil {
					log.Printf("Error handling business message: %v", err)
				}
			}
		}
	}
}

// reconnect attempts to reconnect with exponential backoff
func (c *BusinessClient) reconnect() {
	for attempt := 1; attempt <= c.maxReconnect; attempt++ {
		select {
		case <-c.ctx.Done():
			return
		default:
			delay := c.reconnectDelay * time.Duration(attempt)
			log.Printf("Reconnecting business in %v (attempt %d/%d)", delay, attempt, c.maxReconnect)
			time.Sleep(delay)

			if err := c.Connect(); err != nil {
				log.Printf("Business reconnect attempt %d failed: %v", attempt, err)
				continue
			}

			log.Println("Business reconnected successfully")
			c.resubscribeAll()
			return
		}
	}

	log.Printf("Failed to reconnect business after %d attempts", c.maxReconnect)
}

// Subscribe subscribes to candle channels for instruments
func (c *BusinessClient) Subscribe(instruments []string, channels []string) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	args := make([]map[string]string, 0, len(instruments)*len(channels))

	for _, inst := range instruments {
		for _, ch := range channels {
			args = append(args, map[string]string{
				"channel": ch,
				"instId":  inst,
			})
		}
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

	c.subscribedMu.Lock()
	for _, inst := range instruments {
		c.subscribed[inst] = true
	}
	c.subscribedMu.Unlock()

	log.Printf("Subscribed to instruments: %v with channels: %v", instruments, channels)
	return nil
}

// Unsubscribe unsubscribes from candle channels
func (c *BusinessClient) Unsubscribe(instruments []string, channels []string) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	args := make([]map[string]string, 0, len(instruments)*len(channels))

	for _, inst := range instruments {
		for _, ch := range channels {
			args = append(args, map[string]string{
				"channel": ch,
				"instId":  inst,
			})
		}
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

	c.subscribedMu.Lock()
	for _, inst := range instruments {
		delete(c.subscribed, inst)
	}
	c.subscribedMu.Unlock()

	log.Printf("Unsubscribed from instruments: %v with channels: %v", instruments, channels)
	return nil
}

// GetSubscribed returns the list of currently subscribed instruments
func (c *BusinessClient) GetSubscribed() []string {
	c.subscribedMu.RLock()
	defer c.subscribedMu.RUnlock()

	instruments := make([]string, 0, len(c.subscribed))
	for inst := range c.subscribed {
		instruments = append(instruments, inst)
	}
	return instruments
}

// resubscribeAll resubscribes to all previously subscribed instruments
func (c *BusinessClient) resubscribeAll() {
	instruments := c.GetSubscribed()
	if len(instruments) > 0 {
		log.Printf("Resubscribing business to %d instruments", len(instruments))
		channels := []string{"candle1D", "candle4H", "candle1H", "candle15m"}
		if err := c.Subscribe(instruments, channels); err != nil {
			log.Printf("Failed to resubscribe business: %v", err)
		}
	}
}

// Close gracefully closes the WebSocket connection
func (c *BusinessClient) Close() error {
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
