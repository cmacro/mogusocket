// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package msutil

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	ms "github.com/cmacro/mogusocket"
)

func NewClient(section ms.ClientHandler, addr string, log ms.Logger) *Client {
	return &Client{
		addr:    addr,
		log:     log,
		section: section,
	}
}

// func NewAutoConnectClient(section ms.ClientHandler, addr string, log ms.Logger)

var (
	ErrNoURL            = errors.New("frame socket is no url config")
	ErrAlreadyConnected = errors.New("frame socket is already open")
)

type Client struct {
	addr    string
	log     ms.Logger
	section ms.ClientHandler
}

type ClientAutoConnect struct {
	Client
	conn                net.Conn
	ctx                 context.Context
	AutoReconnectErrors int
}

func (c *ClientAutoConnect) autoReconnect() {
	for {
		autoReconnectDelay := time.Duration(c.AutoReconnectErrors) * 2 * time.Second
		c.log.Debugf("Automatically reconnecting after %v", autoReconnectDelay)
		c.AutoReconnectErrors++
		time.Sleep(autoReconnectDelay)
		err := c.connect()
		if errors.Is(err, ErrAlreadyConnected) {
			c.log.Debugf("Connect() said we're already connected after autoreconnect sleep")
			return
		} else if errors.Is(err, ErrNoURL) {
			c.log.Debugf("Connect() is no url config")
			return
		} else if err != nil {
			c.log.Errorf("Error reconnecting after autoreconnect sleep: %v", err)
		} else {
			return
		}
	}
}

func (c *Client) Run(ctx context.Context) {
	u, _ := ms.ParserAddr(c.addr)
	conn, err := net.Dial(u.Data())
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

	state := ms.StateClientSide
	r := &Reader{Source: conn, State: state, CheckUTF8: true, OnIntermediate: ControlFrameHandler(conn, state)}
	// var buf bytes.Buffer
	w := NewWriter(conn, state, 0)
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
			c.log.Error("connect writer", err)
		}
		return err
	}

	sctx, scancel := context.WithCancel(ctx)
	if err := c.section.Connect(sctx, wh, scancel); err != nil {
		c.log.Error("failed open section", err)
		return
	}

	go func() {
		defer scancel()
		for {
			h, err := r.NextFrame()
			if err != nil {
				if err == io.EOF {
					c.log.Info("socket closed.")
				} else {
					c.log.Error("next frame error", err)
				}
				return
			}
			if h.OpCode.IsControl() {
				c.log.Info("is control", h.OpCode)
				continue
			}
			if err := c.section.ReadDump(r, h.Length, h.OpCode == ms.OpText); err != nil {
				c.log.Info("read dump", err)
				return
			}
		}
	}()

	<-ctx.Done()
}

func (c *Client) work(conn net.Conn, ctx context.Context) {
	defer func() {
		c.log.Debug("client closed.")
		if err := conn.Close(); err != nil {
			c.log.Error("close connection", err)
		}
	}()

	state := ms.StateClientSide
	r := &Reader{Source: conn, State: state, CheckUTF8: true, OnIntermediate: ControlFrameHandler(conn, state)}
	// var buf bytes.Buffer
	w := NewWriter(conn, state, 0)
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
			c.log.Error("connect writer", err)
		}
		return err
	}

	sctx, scancel := context.WithCancel(ctx)
	if err := c.section.Connect(sctx, wh, scancel); err != nil {
		c.log.Error("failed open section", err)
		return
	}

	go func() {
		defer scancel()
		for {
			h, err := r.NextFrame()
			if err != nil {
				if err == io.EOF {
					c.log.Info("socket closed.")
				} else {
					c.log.Error("next frame error", err)
				}
				return
			}
			if h.OpCode.IsControl() {
				c.log.Info("is control", h.OpCode)
				continue
			}
			if err := c.section.ReadDump(r, h.Length, h.OpCode == ms.OpText); err != nil {
				c.log.Info("read dump", err)
				return
			}
		}
	}()

	<-ctx.Done()
}
