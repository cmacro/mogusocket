// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package msutil

import (
	"context"
	"io"

	ms "github.com/cmacro/mogusocket"
)

func NewConnecter(sections ms.SessionsHandler, log ms.Logger) *Connecter {
	return &Connecter{
		log:             log,
		SessionsHandler: sections,
	}
}

type Connecter struct {
	log ms.Logger
	ms.SessionsHandler
}

func (c *Connecter) Run(ctx context.Context, conn io.ReadWriter) {
	sectionCtx, sectionCancel := context.WithCancel(ctx)

	state := ms.StateServerSide
	ch := ControlFrameHandler(conn, state)
	r := &Reader{
		Source:         conn,
		State:          state,
		CheckUTF8:      true,
		OnIntermediate: ch,
	}
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
			sectionCancel()
		}
		return err
	}

	section, err := c.SessionsHandler.Connect(sectionCtx, wh, sectionCancel)
	if err != nil {
		c.log.Info("connection refused", err)
		return
	}
	defer func() {
		c.SessionsHandler.Close(section)
		sectionCancel()
	}()

	for {
		select {
		case <-sectionCtx.Done():
			return

		default:
			h, err := r.NextFrame()
			if err != nil {
				if err == io.EOF {
					c.log.Info("closed", section.GetId())
				} else {
					c.log.Error("next frame error", err)
				}
				return
			}
			if h.OpCode.IsControl() {
				c.log.Debug("is control", h.OpCode)
				if err = ch(h, r); err != nil {
					c.log.Error("handle control", err)
					return
				}
				continue
			}
			err = section.ReadPump(r, h.OpCode == ms.OpText)
			if err != nil {
				c.log.Info("read dump", err)
				return
			}
		}
	}
}
