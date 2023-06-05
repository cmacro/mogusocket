// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

func NewTestSections(log ms.Logger) *Sessions {
	return &Sessions{
		log:   log,
		items: make(map[int64]*Client),
		Mutex: &sync.Mutex{},
	}
}

type Client struct {
	id     int64
	log    ms.Logger
	writer ms.SendFunc
	ctx    context.Context
	cancel func()
}

type Sessions struct {
	log ms.Logger
	*sync.Mutex

	maxid int64
	items map[int64]*Client
}

func (s *Sessions) Connect(ctx context.Context, w ms.SendFunc, c func()) (ms.SessionHandler, error) {
	s.Lock()
	s.maxid++
	nid := s.maxid
	section := &Client{id: nid, writer: w, ctx: ctx, cancel: c, log: s.log.Sub(strconv.FormatInt(nid, 10))}
	s.items[nid] = section
	s.Unlock()

	return section, nil
}

func (s *Sessions) Close(section ms.SessionHandler) error {
	// remove
	id := section.GetId()
	s.Lock()
	delete(s.items, id)
	s.Unlock()
	section.Close()
	return nil
}

func (c *Client) Close() {
	c.cancel()
}

func (c *Client) ReadDump(r io.Reader, isText bool) error {
	b, _ := io.ReadAll(r)
	c.log.Info("read dump", isText, "data:", string(b))
	err := c.writer(strings.NewReader("recv "+string(b)), isText)
	return err
}

func (c *Client) GetId() int64 {
	return c.id
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()
	mainLog = ms.Stdout("Main", "DEBUG", true)

	svrLog := ms.Stdout("Server", "DEBUG", true)
	connecter := msutil.NewConnecter(NewTestSections(ms.Stdout("Sections", "DEBUG", true)), svrLog.Sub("Connect"))
	ms := ms.NewServer(*addr, connecter, svrLog)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ms.Run(ctx)
		cancel()
	}()

	runSysSignal(ctx, cancel)

	wg.Wait()
	<-ctx.Done()

	time.Sleep(1 * time.Second)
	mainLog.Info("closed.")
}
