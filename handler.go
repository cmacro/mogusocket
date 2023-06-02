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

type SectionHandler interface {
	Connect(ctx context.Context, w func(src io.Reader, isText bool) error, c func()) int64
	Close(id int64)
	ReadDump(r io.Reader) error
}
