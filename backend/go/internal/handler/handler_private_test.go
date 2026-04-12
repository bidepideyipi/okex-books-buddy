package handler_test

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/supermancell/okex-buddy/internal/assembly"
	"github.com/supermancell/okex-buddy/internal/handler"
)

func TestAssemblyPrivateWs(t *testing.T) {
	app := assembly.NewAssembly()

	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	msgHandler := handler.NewPrivateMessageHandler(app.MongoClient)
	if err := app.LoadPrivateWs(msgHandler); err != nil {
		t.Fatal(err)
	}

	setupGracefulShutdownAssemblyPrivateWs(app)

	done := make(chan bool, 1)
	go func() {
		time.Sleep(20 * time.Second)
		done <- true
	}()

	<-done
	t.Log("20秒已到，测试结束")

}

func setupGracefulShutdownAssemblyPrivateWs(app *assembly.Assembly) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")

		if err := app.ClosePrivateWs(); err != nil {
			log.Printf("Error closing Private WebSocket: %v", err)
		}

		os.Exit(0)
	}()
}
