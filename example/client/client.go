package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	ms "github.com/cmacro/mogusocket"
	"github.com/cmacro/mogusocket/msutil"
)

var addr = flag.String("listen", "unix:///tmp/ws_testsocket.tmp", "addr to listen")

var mainLog ms.Logger

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

type ClientSession struct {
	*sync.Mutex
	ms.Logger

	writer ms.SendFunc
	ctx    context.Context
	cancel context.CancelFunc
}

func (s *ClientSession) ReadDump(r io.Reader, len int64, isText bool) error {
	payload := make([]byte, len)
	_, err := io.ReadFull(r, payload)

	if err != nil {
		mainLog.Info("read dump", err)
		return err
	}
	mainLog.Info("read payload:", string(payload))
	return nil
}

func (s *ClientSession) Send(txt string) error {
	s.Debug("send text:", txt)
	s.Lock()
	defer s.Unlock()
	return s.writer(strings.NewReader(txt), true)
}

func (s *ClientSession) Connect(ctx context.Context, w ms.SendFunc, c func()) error {
	s.Lock()
	s.writer = w
	s.cancel = c
	s.ctx = ctx
	s.Unlock()

	if err := s.Send("Hello"); err != nil {
		s.cancel()
		return err
	}

	return nil
}

func main() {
	log.SetFlags(log.Lshortfile)
	flag.Parse()

	mainLog = ms.Stdout("Main", "DEBUG", true)

	session := &ClientSession{Mutex: &sync.Mutex{}, Logger: ms.Stdout("Session", "DEBUG", true)}
	clientDial := msutil.NewClient(session, *addr, ms.Stdout("Dial", "DEBUG", true))

	ctx, cancel := context.WithCancel(context.Background())
	go func() { defer cancel(); clientDial.Run(ctx) }()

	go func(session *ClientSession) {
		readlog := mainLog.Sub("read")
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
				if text == ".q" {
					readlog.Info("closed.")
					return
				}
				// strings.Trim()
				readlog.Info("do send:", text)
				if err := session.Send(text); err != nil {
					readlog.Error("send message", err)
					return
				}
			}
		}
	}(session)

	<-ctx.Done()

}
