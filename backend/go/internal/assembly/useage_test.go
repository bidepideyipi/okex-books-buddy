package assembly_test

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/supermancell/okex-buddy/internal/assembly"
)

func TestAssembly(t *testing.T) {
	assembly := assembly.NewAssembly()

	if assembly.Config == nil {
		t.Fatalf("Config is nil")
	}

	t.Log(assembly.Config.Redis.Addr)
	print(assembly.Config.Redis.Addr)

}

func TestLoadRedis(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadRedis(); err != nil {
		t.Fatal(err)
	}

	// 注册优雅关闭
	setupGracefulShutdownRedis(app)

	if app.RedisClient == nil {
		t.Fatalf("RedisClient is nil")
	}
}

func TestLoadMongo(t *testing.T) {
	app := assembly.NewAssembly()
	if err := app.LoadMongo(); err != nil {
		t.Fatal(err)
	}

	// 注册优雅关闭
	setupGracefulShutdownMongo(app)

	if app.MongoClient == nil {
		t.Fatalf("MongoClient is nil")
	}
}

func TestLoadPrivateWs(t *testing.T) {
	app := assembly.NewAssembly()
	// 测试自定义消息处理函数, 实际应用中应根据需要实现消息处理逻辑
	// 这里仅打印收到的消息
	// 消息处理和WS客户端是解耦的，消息处理函数可以独立实现并且传给LoadPrivateWs方法
	if err := app.LoadPrivateWs(func(msg []byte) error {
		log.Printf("Received message: %s", msg)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// 注册优雅关闭
	setupGracefulShutdownPrivateWs(app)

	if app.PrivateClient == nil {
		t.Fatalf("PrivateClient is nil")
	}

	// 创建带缓冲的 channel
	done := make(chan bool, 1)

	// 启动 goroutine 在1分钟后发送信号
	go func() {
		time.Sleep(1 * time.Minute)
		done <- true
	}()

	// 等待1分钟或收到中断信号
	select {
	case <-done:
		t.Log("20秒已到，测试结束")
	case <-time.After(20 * time.Second):
		t.Log("超时，测试结束")
	}
}

// You can setup graceful shutdown in main function like this
func setupGracefulShutdownRedis(app *assembly.Assembly) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")

		if err := app.CloseRedis(); err != nil {
			log.Printf("Error closing Redis: %v", err)
		}

		os.Exit(0)
	}()
}

func setupGracefulShutdownMongo(app *assembly.Assembly) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down gracefully...")

		if err := app.CloseMongo(); err != nil {
			log.Printf("Error closing MongoDB: %v", err)
		}

		os.Exit(0)
	}()
}

func setupGracefulShutdownPrivateWs(app *assembly.Assembly) {
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
