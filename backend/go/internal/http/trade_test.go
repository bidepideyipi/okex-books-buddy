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

	algoOrd := map[string]string{
		"slTriggerPx":     "78.0",
		"slOrdPx":         "77.8",
		"slTriggerPxType": "last",
		"tpTriggerPx":     "85.0",
		"tpOrdPx":         "84.8",
		"tpTriggerPxType": "last",
		// 不填写 slTriggerPx/tpTriggerPx
	}

	resp, err := app.OkxClient.PlaceOrder(&rest.PlaceOrderRequest{
		InstID:         "SOL-USDT-SWAP",
		TdMode:         "cross",
		Side:           "buy",
		PosSide:        "long",
		OrdType:        "limit",
		Px:             "81.0",
		Sz:             "0.1",
		AttachAlgoOrds: []map[string]string{algoOrd},
	})

	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("resp: %v\n", resp)

}
