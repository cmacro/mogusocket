// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package wsutil

import (
	"context"
	"io"

	ws "github.com/cmacro/mogusocket"
)

func NewContainer(section ws.SectionHandler, log ws.Logger) *Container {
	return &Container{
		log:            log,
		SectionHandler: section,
	}
}

type Container struct {
	log ws.Logger
	ws.SectionHandler
}

func (c *Container) Run(ctx context.Context, conn io.ReadWriter) {
	sectionCtx, sectionCancel := context.WithCancel(ctx)

	state := ws.StateServerSide
	ch := ControlFrameHandler(conn, state)
	r := &Reader{
		Source:         conn,
		State:          state,
		CheckUTF8:      true,
		OnIntermediate: ch,
	}
	w := NewWriter(conn, state, 0)
	wh := func(src io.Reader, isText bool) error {
		opcode := ws.OpText
		if !isText {
			opcode = ws.OpBinary
		}
		w.Reset(conn, state, opcode)
		_, err := io.Copy(w, r)
		if err == nil {
			err = w.Flush()
		}
		if err != nil {
			c.log.Error("connect writer", err)
			sectionCancel()
		}
		return err
	}

	sectionid := c.SectionHandler.Connect(sectionCtx, wh)
	defer func() { c.Close(sectionid); sectionCancel() }()

	if sectionid == 0 {
		c.log.Info("connection refused")
		return
	}

	for {
		select {
		case <-sectionCtx.Done():
			return

		default:
			h, err := r.NextFrame()
			if err != nil {
				c.log.Error("next frame error", err)
				return
			}
			if h.OpCode.IsControl() {
				if err = ch(h, r); err != nil {
					c.log.Error("handle control", err)
					return
				}
				continue
			}
			err = c.ReadDump(r)
			if err != nil {
				c.log.Info("read dump", err)
				return
			}
		}
	}
}
