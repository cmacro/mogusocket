package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
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

func main() {
	// log.SetFlags(0)
	flag.Parse()

	mainLog = ms.Stdout("Main", "DEBUG", true)

	// mc := msutil.NewClient(*addr, ms.Stdout("Section", "DEBUG", true))
	// mc.Run()

	u, _ := ms.ParserAddr(*addr)
	conn, err := net.Dial(u.Data())
	if err != nil {
		mainLog.Error("connect", err)
		return
	}
	state := ms.StateClientSide
	r := &msutil.Reader{Source: conn, State: state, CheckUTF8: true, OnIntermediate: msutil.ControlFrameHandler(conn, state)}
	w := msutil.NewWriter(conn, state, 0)
	wh := func(src io.Reader, isText bool) error {
		opcode := ms.OpText
		if !isText {
			opcode = ms.OpBinary
		}
		w.Reset(conn, state, opcode)
		_, err := io.Copy(w, src)
		if err == nil {
			err = w.Flush()
		}
		if err != nil {
			mainLog.Error("connect writer", err)
		}
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		for {
			h, err := r.NextFrame()
			if err != nil {
				if err == io.EOF {
					mainLog.Info("socket closed.")
				} else {
					mainLog.Error("next frame error", err)
				}
				return
			}
			if h.OpCode.IsControl() {
				mainLog.Info("is control", h.OpCode)
				continue
			}
			// err = section.ReadDump(r, h.OpCode == ms.OpText)
			payload := make([]byte, h.Length)
			_, err = io.ReadFull(r, payload)

			if err != nil {
				mainLog.Info("read dump", err)
				return
			}
			mainLog.Info("read payload:", string(payload))
		}
	}()

	readlog := mainLog.Sub("read")

	if err := wh(strings.NewReader("hello"), true); err != nil {
		readlog.Error("failed send message", err)
		return
	}

	go func() {
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
				if err := wh(strings.NewReader(text), true); err != nil {
					readlog.Error("send message", err)
					return
				}
			}
		}
	}()

	<-ctx.Done()

}
