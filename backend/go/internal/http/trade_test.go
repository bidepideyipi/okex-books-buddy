package http_test

import (
	"fmt"
	"testing"

	"github.com/supermancell/okex-buddy/internal/assembly"
	rest "github.com/supermancell/okex-buddy/internal/http"
)

func TestOrderMaker(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	app.LoadOkxRest()

	//如果不自己生成附加域的attachAlgoClOrdId，就要去查一遍pending才能知道attachAlgoOrdId
	attachAlgoClOrdId := rest.GenerateOrderID("ao")
	algoOrd := map[string]string{
		"attachAlgoClOrdId": attachAlgoClOrdId,
		"slTriggerPx":       "78.0", //止损触发价
		"slOrdPx":           "-1",   //止损委托价
		"slTriggerPxType":   "last",
		"tpTriggerPx":       "79.8",
		"tpOrdPx":           "-1",
		"tpTriggerPxType":   "last",
	}

	resp, err := app.OkxClient.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:         "SOL-USDT-SWAP",
		TdMode:         "cross",
		Side:           "buy",
		PosSide:        "long",
		OrdType:        "limit",
		Px:             "79.0",
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

func TestAmendType(t *testing.T) {
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
