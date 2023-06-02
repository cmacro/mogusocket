// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
	log.SetFlags(0)
	flag.Parse()

	mainLog = mogusocket.Stdout("Main", "DEBUG", true)

	ctx, cancel := context.WithCancel(context.Background())
	ws := mogusocket.NewServer(*addr, mogusocket.Stdout("Server", "DEBUG", true))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); ws.Run(ctx) }()

	runSysSignal(ctx, cancel)

	wg.Wait()
	<-ctx.Done()

	time.Sleep(1 * time.Second)
	mainLog.Info("closed.")
	// http.HandleFunc("/ws", wsHandler)
	// http.HandleFunc("/wsutil", wsutilHandler)

}
