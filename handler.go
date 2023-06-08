// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package mogusocket

import (
	"context"
	"io"
)

type ConnectHandler interface {
	Run(ctx context.Context, conn io.ReadWriter)
}

type SendFunc func(src io.Reader, isText bool) error

type SessionHandler interface {
	GetId() int64
	Close()
	ReadPump(r io.Reader, len int64, isText bool) error
}

type ClientHandler interface {
	ReadPump(r io.Reader, len int64, isText bool) error
	Connect(ctx context.Context, w SendFunc, c func()) error
	Close()
}

type SessionsHandler interface {
	Connect(ctx context.Context, w SendFunc, c func()) (SessionHandler, error)
	Close(session SessionHandler) error
	// ReadPump(r io.Reader, isText bool) error
}
