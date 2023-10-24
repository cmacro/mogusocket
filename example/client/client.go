package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	ms "github.com/cmacro/mogusocket"
	"github.com/cmacro/mogusocket/msutil"
)

var (
	addr     = flag.String("listen", "unix:///tmp/ws_testsocket.tmp", "addr to listen")
	autoconn = flag.String("autoconn", "false", "auto connect")
)

var mainLog ms.Logger

// func runSysSignal(ctx context.Context, cancel context.CancelFunc) {
// 	defer mainLog.Info("close sys signal.")
// 	sigs := make(chan os.Signal, 1)
// 	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
// 	go func() {
// 		for {
// 			select {
// 			case <-sigs:
// 				cancel()
// 				return
// 			case <-ctx.Done():
// 				return
// 			}
// 		}
// 	}()
// }

type ClientSession struct {
	*sync.Mutex
	ms.Logger

	writer ms.SendFunc
	ctx    context.Context
	cancel context.CancelFunc
}

func (s *ClientSession) ReadPump(r io.Reader, len int64, isText bool) error {
	payload := make([]byte, len)
	_, err := io.ReadFull(r, payload)

	if err != nil {
		s.Info("read dump", err)
		return err
	}
	s.Info("read payload:", string(payload))
	if isText && string(payload) == "recv cancel" {
		s.Debug("call cancel")
		s.cancel()
	}

	return nil
}

func (s *ClientSession) Send(txt string) error {
	s.Debug("send text:", txt)
	s.Lock()
	defer s.Unlock()
	if s.writer != nil {
		return s.writer(strings.NewReader(txt), true)
	}
	s.Logger.Warn("not connect")
	return nil
}

func (s *ClientSession) Connect(ctx context.Context, w ms.SendFunc, c func()) error {
	s.Info("server connected")

	s.Lock()
	s.writer = w
	s.cancel = c
	s.ctx = ctx
	s.Unlock()

	if err := s.Send("Hello"); err != nil {
		s.Error("failed send", err)
		s.cancel()
		return err
	}
	return nil
}

func (s *ClientSession) Close() {
	s.Info("connect closed.")

	s.Lock()
	defer s.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.writer = nil
	s.cancel = nil
	s.ctx = nil
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	mainLog = ms.Stdout("Main", "DEBUG", true)

	session := &ClientSession{Mutex: &sync.Mutex{}, Logger: ms.Stdout("Session", "DEBUG", true)}
	ctx, cancel := context.WithCancel(context.Background())

	if *autoconn == "true" {
		clientconnect := msutil.NewAutoConnectClient(session, *addr, session.Logger)
		clientconnect.Run(ctx, cancel) // .ConnectServer(*addr, session, ctx, session.Logger)
	} else {
		go msutil.ConnectServer(ctx, *addr, session, cancel, session.Logger)
	}

	go func(session *ClientSession) {
		readlog := ms.Stdout("Read", "DEBUG", true)
		var text string
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, err := fmt.Scanln(&text)
				if err != nil {
					readlog.Error("read os stdin", err)
					return
				}
				select {
				case <-ctx.Done():
					return
				default:
				}

				if text == ".q" {
					readlog.Info("closed.")
					return
				}
				readlog.Info("do send:", text)
				if err := session.Send(text); err != nil {
					readlog.Error("send message", err)
					return
				}
			}
		}
	}(session)

	<-ctx.Done()
	mainLog.Info("closed.")
}
