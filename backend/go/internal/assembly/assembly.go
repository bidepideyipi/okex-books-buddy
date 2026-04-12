package assembly

import (
	"fmt"
	"log"

	"github.com/supermancell/okex-buddy/internal/common"
	"github.com/supermancell/okex-buddy/internal/config"
	rest "github.com/supermancell/okex-buddy/internal/http"
	"github.com/supermancell/okex-buddy/internal/mongodb"
	"github.com/supermancell/okex-buddy/internal/redisclient"
	"github.com/supermancell/okex-buddy/internal/ws"
)

/**
* 通过Assembly定义一个类似Spring的容器，用于管理所有的组件
 */
type Assembly struct {
	Config        *config.AppConfig
	RedisClient   *redisclient.Client
	MongoClient   *mongodb.Client
	PrivateClient *ws.PrivateClient
	OkxClient     *rest.OKExHTTPClient
}

func NewAssembly() *Assembly {
	config := config.LoadFromEnv()
	return &Assembly{Config: &config}
}

func (a *Assembly) LoadRedis() error {
	redisClient, err := redisclient.NewClient(a.Config.Redis.Addr, a.Config.Redis.Password)
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
		a.RedisClient = nil
		return fmt.Errorf("cannot start service without Redis connection: %w", err)
	}

	log.Println("Connected to Redis")
	a.RedisClient = redisClient
	return nil
}

func (a *Assembly) LoadMongo() error {
	mongoClient, err := mongodb.NewClient(a.Config.MongoDB.Addr, a.Config.MongoDB.Database)
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v", err)
		a.MongoClient = nil
		return fmt.Errorf("cannot start service without MongoDB connection: %w", err)
	}

	log.Println("Connected to MongoDB")
	a.MongoClient = mongoClient
	return nil
}

func (a *Assembly) LoadOkxRest() error {
	if a.MongoClient == nil {
		if err := a.LoadMongo(); err != nil {
			return err
		}
	}

	apiKey, secretKey, passphrase, opErr := a.MongoClient.GetOKExConfig()
	if opErr != nil {
		log.Printf("Failed to get OKEx config from MongoDB: %v", opErr)
		log.Printf("Please ensure API credentials are stored in MongoDB config collection")
		return opErr
	}

	privateConfig := ws.OKExConfig{
		APIKey:     apiKey,
		SecretKey:  secretKey,
		Passphrase: passphrase,
	}

	a.OkxClient = rest.NewOKExHTTPClientWithProxy(privateConfig, a.Config.OKEX.UseProxy, a.Config.OKEX.HTTPProxyAddr)

	log.Println("Syncing time with OKEx server for HTTP client...")
	offset, err := a.OkxClient.SyncServerTime()
	if err != nil {
		log.Printf("Warning: Failed to sync time: %v", err)
	} else {
		a.OkxClient.SetTimeOffset(offset)
		log.Printf("HTTP client time offset set to: %d ms", offset)
	}

	return nil
}

func (a *Assembly) LoadPrivateWs(messageHandler common.MessageHandler) error {

	if a.MongoClient == nil {
		if err := a.LoadMongo(); err != nil {
			return err
		}
	}

	apiKey, secretKey, passphrase, opErr := a.MongoClient.GetOKExConfig()
	if opErr != nil {
		log.Printf("Failed to get OKEx config from MongoDB: %v", opErr)
		log.Printf("Please ensure API credentials are stored in MongoDB config collection")
		return opErr
	}

	privateConfig := ws.OKExConfig{
		APIKey:     apiKey,
		SecretKey:  secretKey,
		Passphrase: passphrase,
	}

	var privateClient *ws.PrivateClient
	if a.Config.OKEX.UseProxy {
		privateClient = ws.NewPrivateClientWithDualProxy(a.Config.OKEX.PrivateWSURL, messageHandler, true, a.Config.OKEX.ProxyAddr, a.Config.OKEX.HTTPProxyAddr, privateConfig)
	} else {
		privateClient = ws.NewPrivateClient(a.Config.OKEX.PrivateWSURL, messageHandler, privateConfig)
	}

	if err := privateClient.Connect(); err != nil {
		log.Printf("Failed to connect to Private WebSocket: %v", err)
		return err
	}

	if err := privateClient.Login(); err != nil {
		log.Printf("Failed to login to Private WebSocket: %v", err)
		privateClient.Close()
		return err
	}

	channels := []map[string]string{
		{"channel": "orders", "instType": "SWAP"},
		{"channel": "positions", "instType": "SWAP"},
	}

	if err := privateClient.Subscribe(channels); err != nil {
		log.Printf("Failed to subscribe to private channels: %v", err)
		return err
	}

	log.Println("Private WebSocket connected, authenticated, and subscribed")
	a.PrivateClient = privateClient
	return nil
}

func (a *Assembly) CloseRedis() error {
	if a.RedisClient == nil {
		return nil
	}

	if err := a.RedisClient.Close(); err != nil {
		log.Printf("Failed to close Redis client: %v", err)
		return err
	}

	log.Println("Redis connection closed")
	return nil
}

func (a *Assembly) CloseMongo() error {
	if a.MongoClient == nil {
		return nil
	}

	if err := a.MongoClient.Close(); err != nil {
		log.Printf("Failed to close MongoDB client: %v", err)
		return err
	}

	log.Println("MongoDB connection closed")
	return nil
}

func (a *Assembly) ClosePrivateWs() error {
	if a.PrivateClient == nil {
		return nil
	}

	if err := a.PrivateClient.Close(); err != nil {
		log.Printf("Failed to close Private WebSocket client: %v", err)
		return err
	}

	log.Println("Private WebSocket connection closed")
	return nil
}
