// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package msutil

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	ms "github.com/cmacro/mogusocket"
)

func NewClient(session ms.ClientHandler, addr string, log ms.Logger) *Client {
	return &Client{
		addr:    addr,
		log:     log,
		session: session,
	}
}

func NewAutoConnectClient(session ms.ClientHandler, addr string, log ms.Logger) *AutoConnectClient {
	return &AutoConnectClient{
		addr:    addr,
		log:     log,
		session: session,
		Mutex:   &sync.Mutex{},
	}
}

var (
	ErrNoURL        = errors.New("frame socket is no url config")
	ErrClientClosed = errors.New("client connect closed")
	// ErrAlreadyConnected = errors.New("frame socket is already open")
)

type Client struct {
	addr    string
	log     ms.Logger
	session ms.ClientHandler
}

type AutoConnectClient struct {
	addr string
	log  ms.Logger
	*sync.Mutex
	session             ms.ClientHandler
	ctx                 context.Context
	cancel              context.CancelFunc
	AutoReconnectErrors int
}

func (c *AutoConnectClient) Run(ctx context.Context, cancel context.CancelFunc) {
	c.ctx = ctx
	c.cancel = cancel
	conn, err := DialServer(c.addr)
	if err != nil {
		go c.autoReconnect()
	} else {
		go c.connect(conn)
	}
}

func (c *AutoConnectClient) autoReconnect() {
	var isConnected bool
	defer func() {
		if !isConnected {
			c.cancel()
		}
	}()

	for {
		autoReconnectDelay := time.Duration(c.AutoReconnectErrors) * 2 * time.Second
		c.log.Debug("Automatically reconnecting after", autoReconnectDelay)
		c.AutoReconnectErrors++
		time.Sleep(autoReconnectDelay)

		conn, err := DialServer(c.addr)
		if err != nil {
			if errors.Is(err, ErrNoURL) {
				c.log.Debug("Connect() is no url config")
				return
			} else if err != nil {
				c.log.Error("Error reconnecting after autoreconnect sleep:", err)
			}
		} else {
			go c.connect(conn)
			c.AutoReconnectErrors = 0
			isConnected = true
			return
		}
	}
}

func (c *AutoConnectClient) close(code int) {
	if code == 0 {
		go c.autoReconnect()
	} else {
		c.cancel()
	}
}

func (c *AutoConnectClient) connect(conn net.Conn) {
	var code int
	defer func() {
		conn.Close()
		go c.close(code)
	}()

	connClosed := make(chan error, 1)
	go func() { connClosed <- ConnectClient(c.ctx, conn, c.session, c.log) }()

	select {
	case <-c.ctx.Done():
		c.log.Debug("context done")
		code = 1 // dont connect
	case err := <-connClosed:
		c.log.Debug("ConnectClient done")
		if err != nil {
			if err == ErrClientClosed {
				c.log.Info("client request closed")
				code = 1
			} else if err == io.EOF {
				c.log.Info("server closed.")
			} else {
				c.log.Error("connect client", err)
			}
		}
	}
}

func (c *Client) Run(ctx context.Context) {
	conn, err := DialServer(c.addr)
	if err != nil {
		c.log.Error("connect", err)
		return
	}
	c.log.Debug("client dial", c.addr)
	defer func() {
		c.log.Debug("client closed.")
		if err := conn.Close(); err != nil {
			c.log.Error("close connection", err)
		}
	}()
	ConnectClient(ctx, conn, c.session, c.log)
}

func ConnectServer(ctx context.Context, addr string, session ms.ClientHandler, log ms.Logger) error {
	conn, err := DialServer(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = ConnectClient(ctx, conn, session, log)
	return err
}

func DialServer(addr string) (net.Conn, error) {
	u, _ := ms.ParserAddr(addr)
	return net.Dial(u.Data())
}

func ConnectClient(ctx context.Context, conn net.Conn, session ms.ClientHandler, log ms.Logger) error {

	state := ms.StateClientSide
	r := &Reader{Source: conn, State: state, CheckUTF8: true, OnIntermediate: ControlFrameHandler(conn, state)}
	w := NewWriter(conn, state, 0)

	writehandler := func(src io.Reader, isText bool) error {
		opcode := ms.OpText
		if !isText {
			opcode = ms.OpBinary
		}
		w.Reset(conn, state, opcode)
		_, err := io.Copy(w, src)
		if err == nil {
			err = w.Flush()
		}
		return err
	}

	sctx, scancel := context.WithCancel(ctx)
	if err := session.Connect(sctx, writehandler, scancel); err != nil {
		log.Error("failed open section", err)
		return err
	}
	defer func() {
		scancel()
		session.Close()
	}()

	go func() {
		<-sctx.Done()
		conn.Close()
	}()

	for {
		h, err := r.NextFrame()
		if err != nil {
			return err
		}
		if h.OpCode.IsControl() {
			log.Info("is control", h.OpCode)
			continue
		}
		if err := session.ReadPump(r, h.Length, h.OpCode == ms.OpText); err != nil {
			log.Info("read dump", err)
			return err
		}
	}
}
