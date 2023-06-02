// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package mogusocket

import (
	"context"
	"net"
	"net/url"
	"os"
	"strings"
)

func NewServer(addr string, log Logger) *Server {
	return &Server{
		addr:   addr,
		Logger: log,
	}
}

type Server struct {
	Logger
	addr string
	ConnectHandler
}

func clearEnvConnect(scheme, path string) error {
	if scheme == "unix" {
		err := os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (s *Server) Run(ctx context.Context) {

	u, err := url.Parse(s.addr)
	if err != nil {
		s.Error("failed addr parser ", s.addr, err)
		return
	}
	if err := clearEnvConnect(u.Scheme, u.Path); err != nil {
		s.Error("Error removing socket file", err)
		return
	}

	listener, err := net.Listen(u.Scheme, u.Path)
	if err != nil {
		s.Error("failed net listen ", s.addr, err)
		return
	}
	s.Info("listening :", s.addr)
	defer func() {
		err := listener.Close()
		if err != nil {
			s.Error("listener closed", err)
		}
		_ = clearEnvConnect(u.Scheme, u.Path)
	}()
	go s.handleAccept(ctx, listener)

	<-ctx.Done()
	s.Info("Server closed", s.addr)
}

func (s *Server) handleAccept(ctx context.Context, ln net.Listener) {
	defer s.Info("listener closed.")
	for {
		select {
		case <-ctx.Done():
			s.Info("stop handle accept.")
			return
		default:
			conn, err := ln.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					s.Info("Accept closed.")
				} else {
					s.Warn("handle accept failure", err)
				}
				continue
			}
			s.Info("conn open", conn.RemoteAddr().String())
			go func() {
				defer func() {
					if err := conn.Close(); err != nil {
						s.Error("conn close error.", err)
					} else {
						s.Info("conn close", conn.RemoteAddr().String())
					}
				}()
				s.handleConnection(ctx, conn)
			}()
		}
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	// dst Writer, src Reader

	<-ctx.Done()
}
