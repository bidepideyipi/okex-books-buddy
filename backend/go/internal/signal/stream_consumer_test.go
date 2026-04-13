package signal_test

import (
	"log"
	"os"
	"syscall"
	"testing"
	"time"

	os_signal "os/signal"

	"github.com/supermancell/okex-buddy/internal/assembly"
	"github.com/supermancell/okex-buddy/internal/common"
	"github.com/supermancell/okex-buddy/internal/signal"
)

func TestStreamConsumer(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}
	if err := app.LoadRedis(); err != nil {
		t.Fatal(err)
	}

	signalConsumer := signal.NewStreamConsumer(app.RedisClient.Client(), func(sig *common.StreamSignal) error {
		log.Printf("Received stream signal: %+v", sig)
		return nil
	})
	signalConsumer.Start()

	setupGracefulShutdown(app)

	done := make(chan bool, 1)
	go func() {
		time.Sleep(20 * time.Second)
		done <- true
	}()

	<-done
	t.Log("20秒已到，测试结束")
}

func setupGracefulShutdown(app *assembly.Assembly) {
	c := make(chan os.Signal, 1)
	os_signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")

		app.CloseStreamConsumer()

		os.Exit(0)
	}()
}
