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

	ms "github.com/cmacro/mogusocket"
	"github.com/cmacro/mogusocket/msutil"
)

var addr = flag.String("listen", "unix:///tmp/ws_testsocket.tmp", "addr to listen")

var mainLog ms.Logger

func runSysSignal(ctx context.Context, cancel context.CancelFunc) {
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
	mainLog = ms.Stdout("Main", "DEBUG", true)

	svrLog := ms.Stdout("Server", "DEBUG", true)
	connecter := msutil.NewConnecter(NewTestSections(ms.Stdout("Sections", "DEBUG", true)), svrLog.Sub("Connect"))
	ws := ms.NewServer(*addr, connecter, svrLog)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); ws.Run(ctx) }()

	runSysSignal(ctx, cancel)

	wg.Wait()
	<-ctx.Done()

	time.Sleep(1 * time.Second)
	mainLog.Info("closed.")
}
