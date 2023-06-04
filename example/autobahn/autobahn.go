package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ms "github.com/cmacro/mogusocket"
	"github.com/cmacro/mogusocket/msutil"
	"github.com/gobwas/httphead"
)

var addr = flag.String("listen", ":9002", "addr to listen")

func main() {
	log.SetFlags(0)
	flag.Parse()

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/wsutil", wsutilHandler)
	http.HandleFunc("/helpers/low", helpersLowLevelHandler)
	http.HandleFunc("/helpers/high", helpersHighLevelHandler)

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %q error: %v", *addr, err)
	}
	log.Printf("listening %s (%q)", ln.Addr(), *addr)

	var (
		s     = new(http.Server)
		serve = make(chan error, 1)
		sig   = make(chan os.Signal, 1)
	)
	signal.Notify(sig, syscall.SIGTERM)
	go func() { serve <- s.Serve(ln) }()

	select {
	case err := <-serve:
		log.Fatal(err)
	case sig := <-sig:
		const timeout = 5 * time.Second

		log.Printf("signal %q received; shutting down with %s timeout", sig, timeout)

		ctx, _ := context.WithTimeout(context.Background(), timeout)
		if err := s.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}
}

var (
	closeInvalidPayload = ms.MustCompileFrame(
		ms.NewCloseFrame(ms.NewCloseFrameBody(
			ms.StatusInvalidFramePayloadData, "",
		)),
	)
	closeProtocolError = ms.MustCompileFrame(
		ms.NewCloseFrame(ms.NewCloseFrameBody(
			ms.StatusProtocolError, "",
		)),
	)
)

func helpersHighLevelHandler(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ms.UpgradeHTTP(r, w)
	if err != nil {
		log.Printf("upgrade error: %s", err)
		return
	}
	defer conn.Close()

	for {
		bts, op, err := msutil.ReadClientData(conn)
		if err != nil {
			log.Printf("read message error: %v", err)
			return
		}
		err = msutil.WriteServerMessage(conn, op, bts)
		if err != nil {
			log.Printf("write message error: %v", err)
			return
		}
	}
}

func helpersLowLevelHandler(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ms.UpgradeHTTP(r, w)
	if err != nil {
		log.Printf("upgrade error: %s", err)
		return
	}
	defer conn.Close()

	msg := make([]msutil.Message, 0, 4)

	for {
		msg, err = msutil.ReadClientMessage(conn, msg[:0])
		if err != nil {
			log.Printf("read message error: %v", err)
			return
		}
		for _, m := range msg {
			if m.OpCode.IsControl() {
				err := msutil.HandleClientControlMessage(conn, m)
				if err != nil {
					log.Printf("handle control error: %v", err)
					return
				}
				continue
			}
			err := msutil.WriteServerMessage(conn, m.OpCode, m.Payload)
			if err != nil {
				log.Printf("write message error: %v", err)
				return
			}
		}
	}
}

func wsutilHandler(res http.ResponseWriter, req *http.Request) {
	conn, _, _, err := ms.UpgradeHTTP(req, res)
	if err != nil {
		log.Printf("upgrade error: %s", err)
		return
	}
	defer conn.Close()

	state := ms.StateServerSide

	ch := msutil.ControlFrameHandler(conn, state)
	r := &msutil.Reader{
		Source:         conn,
		State:          state,
		CheckUTF8:      true,
		OnIntermediate: ch,
	}
	w := msutil.NewWriter(conn, state, 0)

	for {
		h, err := r.NextFrame()
		if err != nil {
			log.Printf("next frame error: %v", err)
			return
		}
		if h.OpCode.IsControl() {
			if err = ch(h, r); err != nil {
				log.Printf("handle control error: %v", err)
				return
			}
			continue
		}

		w.Reset(conn, state, h.OpCode)

		if _, err = io.Copy(w, r); err == nil {
			err = w.Flush()
		}
		if err != nil {
			log.Printf("echo error: %s", err)
			return
		}
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	u := ms.HTTPUpgrader{
		Extension: func(opt httphead.Option) bool {
			log.Printf("extension: %s", opt)
			return false
		},
	}
	conn, _, _, err := u.Upgrade(r, w)
	if err != nil {
		log.Printf("upgrade error: %s", err)
		return
	}
	defer conn.Close()

	state := ms.StateServerSide

	textPending := false
	utf8Reader := msutil.NewUTF8Reader(nil)
	cipherReader := msutil.NewCipherReader(nil, [4]byte{0, 0, 0, 0})

	for {
		header, err := ms.ReadHeader(conn)
		if err != nil {
			log.Printf("read header error: %s", err)
			break
		}
		if err = ms.CheckHeader(header, state); err != nil {
			log.Printf("header check error: %s", err)
			conn.Write(closeProtocolError)
			return
		}

		cipherReader.Reset(
			io.LimitReader(conn, header.Length),
			header.Mask,
		)

		var utf8Fin bool
		var r io.Reader = cipherReader

		switch header.OpCode {
		case ms.OpPing:
			header.OpCode = ms.OpPong
			header.Masked = false
			ms.WriteHeader(conn, header)
			io.CopyN(conn, cipherReader, header.Length)
			continue

		case ms.OpPong:
			io.CopyN(io.Discard, conn, header.Length)
			continue

		case ms.OpClose:
			utf8Fin = true

		case ms.OpContinuation:
			if textPending {
				utf8Reader.Source = cipherReader
				r = utf8Reader
			}
			if header.Fin {
				state = state.Clear(ms.StateFragmented)
				textPending = false
				utf8Fin = true
			}

		case ms.OpText:
			utf8Reader.Reset(cipherReader)
			r = utf8Reader

			if !header.Fin {
				state = state.Set(ms.StateFragmented)
				textPending = true
			} else {
				utf8Fin = true
			}

		case ms.OpBinary:
			if !header.Fin {
				state = state.Set(ms.StateFragmented)
			}
		}

		payload := make([]byte, header.Length)
		_, err = io.ReadFull(r, payload)
		if err == nil && utf8Fin && !utf8Reader.Valid() {
			err = msutil.ErrInvalidUTF8
		}
		if err != nil {
			log.Printf("read payload error: %s", err)
			if err == msutil.ErrInvalidUTF8 {
				conn.Write(closeInvalidPayload)
			} else {
				conn.Write(ms.CompiledClose)
			}
			return
		}

		if header.OpCode == ms.OpClose {
			code, reason := ms.ParseCloseFrameData(payload)
			log.Printf("close frame received: %v %v", code, reason)

			if !code.Empty() {
				switch {
				case code.IsProtocolSpec() && !code.IsProtocolDefined():
					err = fmt.Errorf("close code from spec range is not defined")
				default:
					err = ms.CheckCloseFrameData(code, reason)
				}
				if err != nil {
					log.Printf("invalid close data: %s", err)
					conn.Write(closeProtocolError)
				} else {
					ms.WriteFrame(conn, ms.NewCloseFrame(ms.NewCloseFrameBody(
						code, "",
					)))
				}
				return
			}

			conn.Write(ms.CompiledClose)
			return
		}

		header.Masked = false
		ms.WriteHeader(conn, header)
		conn.Write(payload)
	}
}
