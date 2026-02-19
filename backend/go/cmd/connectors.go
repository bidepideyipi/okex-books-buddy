package main

import (
	"log"

	"github.com/supermancell/okex-buddy/internal/config"
	httpserver "github.com/supermancell/okex-buddy/internal/http"
	"github.com/supermancell/okex-buddy/internal/handler"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/orderbook"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	"github.com/supermancell/okex-buddy/internal/signal"
	"github.com/supermancell/okex-buddy/internal/ws"
)

// ConnectPublicWebSocket connects to the public WebSocket endpoint
func ConnectPublicWebSocket(cfg config.AppConfig, obManager *orderbook.Manager) *ws.PublicClient {
	log.Printf("Public WebSocket is enabled, connecting to: %s", cfg.OKEX.PublicWSURL)
	messageHandler := handler.NewPublicMessageHandler(obManager)

	var wsClient *ws.PublicClient
	if cfg.OKEX.UseProxy {
		log.Printf("Proxy enabled: %s", cfg.OKEX.ProxyAddr)
		wsClient = ws.NewPublicClientWithProxy(cfg.OKEX.PublicWSURL, messageHandler, true, cfg.OKEX.ProxyAddr)
	} else {
		log.Println("Proxy disabled, connecting directly")
		wsClient = ws.NewPublicClient(cfg.OKEX.PublicWSURL, messageHandler)
	}

	if err := wsClient.Connect(); err != nil {
		log.Printf("Failed to connect to OKEx WebSocket: %v", err)
		httpserver.SetWSHealthy(false)
		return nil
	}

	log.Println("Connected to OKEx WebSocket")
	return wsClient
}

// ConnectBusinessWebSocket connects to the business WebSocket endpoint
func ConnectBusinessWebSocket(cfg config.AppConfig, mongoClient *mongodb.Client) *ws.BusinessClient {
	log.Printf("Business WebSocket is enabled, connecting to: %s", cfg.OKEX.BusinessWSURL)
	businessMessageHandler := handler.NewBusinessMessageHandler(mongoClient)

	var businessWsClient *ws.BusinessClient
	if cfg.OKEX.UseProxy {
		log.Printf("Business WebSocket using proxy: %s", cfg.OKEX.ProxyAddr)
		businessWsClient = ws.NewBusinessClientWithProxy(cfg.OKEX.BusinessWSURL, businessMessageHandler, true, cfg.OKEX.ProxyAddr)
	} else {
		log.Println("Business WebSocket proxy disabled, connecting directly")
		businessWsClient = ws.NewBusinessClient(cfg.OKEX.BusinessWSURL, businessMessageHandler)
	}

	log.Println("Attempting to connect to Business WebSocket...")
	if err := businessWsClient.Connect(); err != nil {
		log.Printf("Failed to connect to OKEx Business WebSocket: %v", err)
		return nil
	}

	log.Println("Connected to OKEx Business WebSocket")

	instruments := []string{"ETH-USDT-SWAP"}
	log.Printf("Subscribing to Business WebSocket instruments: %v", instruments)
	if err := businessWsClient.Subscribe(instruments); err != nil {
		log.Printf("Failed to subscribe to candlestick channels: %v", err)
	} else {
		log.Println("Successfully subscribed to candlestick channels")
	}

	return businessWsClient
}

// ConnectPrivateWebSocket connects to OKEx private WebSocket
func ConnectPrivateWebSocket(cfg config.AppConfig, mongoClient *mongodb.Client, redisClient *redisclient.Client) *ws.PrivateClient {
	log.Println("Connecting to Private WebSocket...")

	apiKey, secretKey, passphrase, err := mongoClient.GetOKExConfig()
	if err != nil {
		log.Printf("Failed to get OKEx config from MongoDB: %v", err)
		log.Printf("Please ensure API credentials are stored in MongoDB config collection")
		return nil
	}

	privateConfig := ws.OKExConfig{
		APIKey:     apiKey,
		SecretKey:  secretKey,
		Passphrase: passphrase,
	}

	orderProcessor := signal.NewOrderProcessor(nil, mongoClient)
	msgHandler := handler.NewPrivateMessageHandler(mongoClient, orderProcessor)

	var privateClient *ws.PrivateClient
	if cfg.OKEX.UseProxy {
		privateClient = ws.NewPrivateClientWithDualProxy(cfg.OKEX.PrivateWSURL, msgHandler, true, cfg.OKEX.ProxyAddr, cfg.OKEX.HTTPProxyAddr, privateConfig)
	} else {
		privateClient = ws.NewPrivateClient(cfg.OKEX.PrivateWSURL, msgHandler, privateConfig)
	}

	orderProcessor = signal.NewOrderProcessor(privateClient, mongoClient)

	if err := privateClient.Connect(); err != nil {
		log.Printf("Failed to connect to Private WebSocket: %v", err)
		return nil
	}

	if err := privateClient.Login(); err != nil {
		log.Printf("Failed to login to Private WebSocket: %v", err)
		privateClient.Close()
		return nil
	}

	channels := []map[string]string{
		{"channel": "orders", "instType": "SWAP"},
		{"channel": "positions", "instType": "SWAP"},
		{"channel": "account-greeks", "instType": "SWAP"},
	}

	if err := privateClient.Subscribe(channels); err != nil {
		log.Printf("Failed to subscribe to private channels: %v", err)
		return nil
	}

	log.Println("Private WebSocket connected, authenticated, and subscribed")

	return privateClient
}
