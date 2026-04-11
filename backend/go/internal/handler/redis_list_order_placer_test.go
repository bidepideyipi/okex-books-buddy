package handler_test

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/supermancell/okex-buddy/internal/assembly"
	common "github.com/supermancell/okex-buddy/internal/common"
	"github.com/supermancell/okex-buddy/internal/handler"
)

// 下单是可以独立于handler_redis_list和signal_consumer被调用的，这是我想要的解耦效果
func TestOrderMaker(t *testing.T) {
	app := assembly.NewAssembly()
	// 测试自定义消息处理函数, 实际应用中应根据需要实现消息处理逻辑
	// 这里仅打印收到的消息
	// 消息处理和WS客户端是解耦的，消息处理函数可以独立实现并且传给LoadPrivateWs方法
	if err := app.LoadPrivateWs(func(msg []byte) error {
		//log.Printf("Received message: %s", msg)
		//下单成功的消息体是这样的：
		// {"id":"1775897795764","op":"order","code":"0","msg":"","data":[{"tag":"","ts":"1775897796026","ordId":"3469373785093513216","clOrdId":"1775897795764","sCode":"0","sMsg":"Order successfully placed.","subCode":""}],"inTime":"1775897796026175","outTime":"1775897796027785"}
		// 订单状态更新的消息体是这样的：
		// {"arg":{"channel":"orders","instType":"SWAP","uid":"244780794298642432"},"data":[{"instType":"SWAP","instId":"ETH-USDT-SWAP","tgtCcy":"","ccy":"USDT","tradeQuoteCcy":"","ordId":"3469373785093513216","clOrdId":"1775897795764","algoClOrdId":"","algoId":"","tag":"","px":"2250","sz":"0.1","notionalUsd":"22.508325","ordType":"limit","side":"buy","posSide":"long","tdMode":"cross","accFillSz":"0","fillNotionalUsd":"","avgPx":"0","state":"live","lever":"10","pnl":"0","feeCcy":"USDT","fee":"0","rebateCcy":"USDT","rebate":"0","category":"normal","uTime":"1775897796026","cTime":"1775897796026","source":"","reduceOnly":"false","cancelSource":"","quickMgnType":"","stpId":"","stpMode":"cancel_maker","attachAlgoClOrdId":"","lastPx":"2232.72","outcome":"","isTpLimit":"false","slTriggerPx":"","slTriggerPxType":"","tpOrdPx":"","tpTriggerPx":"","tpTriggerPxType":"","slOrdPx":"","fillPx":"","tradeId":"","fillSz":"0","fillTime":"","fillPnl":"0","fillFee":"0","fillFeeCcy":"","execType":"","fillPxVol":"","fillPxUsd":"","fillMarkVol":"","fillFwdPx":"","fillMarkPx":"","fillIdxPx":"","amendSource":"","reqId":"","amendResult":"","code":"0","msg":"","pxType":"","pxUsd":"","pxVol":"","linkedAlgoOrd":{"algoId":""},"attachAlgoOrds":[]}]}
		//事件的消息体是
		//{"event":"subscribe","arg":{"channel":"orders","instType":"SWAP"},"connId":"0b38c7cc"}
		//{"event":"channel-conn-count","channel":"orders","connCount":"1","connId":"0b38c7cc"}
		// 打印消息体
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

	orderMaker := handler.NewOrderMaker(app.PrivateClient)

	clOrdID, ordID, err := orderMaker.PlaceOrder(&common.Signal{
		SignalID: "test",
		InstID:   "10461", //ETH-USDT-SWAP
		Side:     "buy",
		OrdType:  "limit",
		PosSide:  "long",
		Sz:       "0.1",
		Px:       "2250",
	})

	if err != nil {
		t.Fatal(err)
	}

	// 打印clOrdID
	t.Log("clOrdID: " + clOrdID)
	// 打印ordID
	t.Log("ordID: " + ordID)

	// 创建带缓冲的 channel
	done := make(chan bool, 1)

	// 启动 goroutine 在1分钟后发送信号
	go func() {
		time.Sleep(20 * time.Second)
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
