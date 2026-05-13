package mongodb_test

import (
	"testing"

	"github.com/supermancell/okex-buddy/internal/mongodb"
)

func TestConnect(t *testing.T) {
	mongoClient := setupMongoDB(t)
	defer teardownMongoDB(t, mongoClient)

	t.Log("The connection to MongoDB was established successfully.")
	// 这里可以添加实际的测试逻辑
}

func TestGetActivePositions(t *testing.T) {
	mongoClient := setupMongoDB(t)
	defer teardownMongoDB(t, mongoClient)

	positions, err := mongoClient.GetActivePositions()
	if err != nil {
		t.Fatalf("Failed to get active positions from MongoDB: %v", err)
	}
	t.Log("Active positions:", positions)

	var posSz float64 = 0
	var posSide string = ""
	var instID string = "ETH-USDT-SWAP"

	for _, p := range positions {
		//判断是否是当前InstID的Position
		if p.InstID != instID {
			continue
		}

		posSide = p.PosSide
		posSz += p.Pos
		break
	}

	t.Log("Active position:", posSide, posSz)
}

func setupMongoDB(t *testing.T) *mongodb.Client {
	t.Helper()
	client, err := mongodb.NewClient("mongodb://127.0.0.1:27017", "technical_analysis")
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	return client
}

func teardownMongoDB(t *testing.T, client *mongodb.Client) {
	t.Helper()
	if client == nil {
		return
	}
	if err := client.Close(); err != nil {
		t.Errorf("Failed to close MongoDB client: %v", err)
	}
}
