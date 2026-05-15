package http_test

import (
	"fmt"
	"testing"

	"github.com/supermancell/okex-buddy/internal/assembly"
	rest "github.com/supermancell/okex-buddy/internal/http"
)

// 市价开多 和现在handler_redis_stream.go 中的openOrder 保持一致
func TestOpenOrder(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	app.LoadOkxRest()

	resp, err := app.OkxClient.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:  "ETH-USDT-SWAP",
		TdMode:  "cross",
		Side:    "buy",
		PosSide: "long",
		OrdType: "market",
		//Px:             strconv.FormatFloat(price, 'f', 2, 64),
		Sz: "0.1",
		//AttachAlgoOrds: []map[string]string{algoOrd},
	})

	if err != nil {
		t.Fatal(err)
	}

	//resp: &{0  [{3472763460055277568   0 Order placed}]}
	fmt.Printf("resp: %v\n", resp)
}

// 市价开空 和现在handler_redis_stream.go 中的openOrder 保持一致
func TestOpenOrderShort(t *testing.T) {
	app := assembly.NewAssembly()
	// if err := app.LoadMongo(); err != nil {
	// 	t.Fatal(err)
	// }

	app.LoadOkxRest()

	resp, err := app.OkxClient.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:  "ETH-USDT-SWAP",
		TdMode:  "cross",
		Side:    "sell",
		PosSide: "short",
		OrdType: "market",
		//Px:             strconv.FormatFloat(price, 'f', 2, 64),
		Sz: "0.1",
		//AttachAlgoOrds: []map[string]string{algoOrd},
	})

	if err != nil {
		t.Fatal(err)
	}

	//resp: &{0  [{3472763460055277568   0 Order placed}]}
	fmt.Printf("resp: %v\n", resp)
}

func TestOrderPlace(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	app.LoadOkxRest()

	//如果不自己生成附加域的attachAlgoClOrdId，就要去查一遍pending才能知道attachAlgoOrdId
	attachAlgoClOrdId := rest.GenerateOrderID("ao")
	algoOrd := map[string]string{
		"attachAlgoClOrdId": attachAlgoClOrdId,
		"slTriggerPx":       "2176.0", //止损触发价
		"slOrdPx":           "-1",     //止损委托价
		"slTriggerPxType":   "last",
		"tpTriggerPx":       "2196.0",
		"tpOrdPx":           "-1",
		"tpTriggerPxType":   "last",
	}

	resp, err := app.OkxClient.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:         "ETH-USDT-SWAP",
		TdMode:         "cross",
		Side:           "buy",
		PosSide:        "long",
		OrdType:        "limit",
		Px:             "2186.0",
		Sz:             "0.1",
		AttachAlgoOrds: []map[string]string{algoOrd},
	})

	if err != nil {
		t.Fatal(err)
	}

	//resp: &{0  [{3472763460055277568   0 Order placed}]}
	fmt.Printf("resp: %v\n", resp)
	fmt.Printf("attachAlgoClOrdId: %v\n", attachAlgoClOrdId)

}

// 市价平多
func TestSellMarket(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	app.LoadOkxRest()

	resp, err := app.OkxClient.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:  "ETH-USDT-SWAP",
		TdMode:  "cross",
		Side:    "sell",
		PosSide: "long",
		OrdType: "market",
		Sz:      "0.1",
	})

	if err != nil {
		t.Fatal(err)
	}

	//resp: &{0  [{3472763460055277568   0 Order placed}]}
	fmt.Printf("resp: %v\n", resp)

}

// 26.04.03 测试了使用止盈止损单功能，成交后的止盈止损单不会在pending中显示
// 也就是实际订单数量是Position + PendingOrder
func TestOrdersPending(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	app.LoadOkxRest()

	resp, err := app.OkxClient.OrdersPending(&rest.OrdersPendingRequest{
		InstType: "SWAP",
	})

	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("resp: %v\n", resp)
}

// 平仓限价订单
func TestTriggerSell(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	app.LoadOkxRest()

	//https://www.okx.com/docs-v5/zh/#order-book-trading-algo-trading-post-place-algo-order
	resp, err := app.OkxClient.PlaceAlgoOrder(&rest.PlaceAlgoOrderRequest{
		InstID:      "ETH-USDT-SWAP",
		TdMode:      "cross",
		Side:        "sell",
		PosSide:     "long",
		OrdType:     "conditional",
		Sz:          "0.2",
		TpTriggerPx: "2450",
		TpOrdPx:     "-1",
		SlTriggerPx: "2300",
		SlOrdPx:     "-1",
		//二者只能写一个，同时写取其后者
	})

	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("resp: %v\n", resp)
}

func TestCancelOrder(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	app.LoadOkxRest()

	resp, err := app.OkxClient.CancelOrder(&rest.CancelOrderRequest{
		InstID: "SOL-USDT-SWAP",
		OrdId:  "3472257774598823936",
	})

	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("resp: %v\n", resp)
}

func TestAmendOrder(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	app.LoadOkxRest()

	resp, err := app.OkxClient.AmendOrder(&rest.AmendOrderRequest{
		InstID: "SOL-USDT-SWAP",
		OrdId:  "3472890636016623616",
		//NewSz:  "0.2",
		NewPx: "78.8",
		AttachAlgoOrds: []map[string]string{
			{
				"attachAlgoClOrdId": "ao1776002604603v",
				"newSlTriggerPx":    "77.8", //止损触发价
				"newTpTriggerPx":    "79.8",
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("resp: %v\n", resp)
}
