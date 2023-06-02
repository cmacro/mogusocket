package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/cmacro/mogusocket"
)

var addr = flag.String("listen", "unix:///tmp/ws_testsocket.tmp", "addr to listen")

var mainLog mogusocket.Logger

func runSysSignal(ctx context.Context, cancel context.CancelFunc) {
	defer mainLog.Info("close sys signal.")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			select {
			case <-sigs:
				cancel()
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func main() {
	flag.Parse()

	mainLog = mogusocket.Stdout("Main", "DEBUG", true)

	ctx, cancel := context.WithCancel(context.Background())
	u, _ := mogusocket.ParserAddr(*addr)
	conn, err := net.Dial(u.Network, u.Address)
	if err != nil {
		mainLog.Error("connect", err)
		return
	}

	// ws := mogusocket.NewServer(*addr, mogusocket.Stdout("Server", "DEBUG", true))

	// var wg sync.WaitGroup
	// wg.Add(1)
	// go func() { defer wg.Done(); ws.Run(ctx) }()

	// runSysSignal(ctx, cancel)

	// wg.Wait()
	// <-ctx.Done()

	// time.Sleep(1 * time.Second)
	// mainLog.Info("closed.")
	// // http.HandleFunc("/ws", wsHandler)
	// // http.HandleFunc("/wsutil", wsutilHandler)

}
