package signal_test

import (
	"testing"
)

func TestOrderMake(t *testing.T) {

	print("Hello World")
	// /**
	//  * 连接到 MongoDB
	//  */
	// var mongoClient *mongodb.Client
	// mongoClient, _ = mongodb.NewClient("mongodb://127.0.0.1:27017", "technical_analysis")
	// defer func() {
	// 	if err := mongoClient.Close(); err != nil {
	// 		log.Printf("Failed to close MongoDB client: %v", err)
	// 	}
	// }()

	/**
	 * 连接到 Private WebSocket
	 */
	//var privateWsClient *ws.PrivateClient

	// orderMaker := signal.NewOrderMaker(privateClient)
	// orderMaker.PlaceOrder(&signal.Signal{
	// 	InstID: "ETH-USDT-SWAP",
	// 	Prediction: "buy",
	// 	Price:    "10000",
	// })
}
