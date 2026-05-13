package handler_test

import (
	"testing"

	"github.com/supermancell/okex-buddy/internal/assembly"
	"github.com/supermancell/okex-buddy/internal/handler"
)

func TestCloseOrder(t *testing.T) {
	app := assembly.NewAssembly()
	app.LoadOkxRest()
	handler.CloseOrder(app.OkxClient, "ETH-USDT-SWAP", "long", "0.1")
}
