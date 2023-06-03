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

type SectionHandler interface {
	GetId() int64
	Close()
	ReadDump(r io.Reader, isText bool) error
}

type SectionsHandler interface {
	Connect(ctx context.Context, w SendFunc, c func()) (SectionHandler, error)
	Close(section SectionHandler) error
	// ReadDump(r io.Reader, isText bool) error
}
