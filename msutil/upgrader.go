// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package msutil

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	ms "github.com/cmacro/mogusocket"
)

// DebugUpgrader is a wrapper around ms.Upgrader. It tracks I/O of a
// WebSocket handshake.
//
// Note that it must not be used in production applications that requires
// Upgrade() to be efficient.
type DebugUpgrader struct {
	// Upgrader contains upgrade to WebSocket options.
	Upgrader ms.Upgrader

	// OnRequest and OnResponse are the callbacks that will be called with the
	// HTTP request and response respectively.
	OnRequest, OnResponse func([]byte)
}

// Upgrade calls Upgrade() on underlying ms.Upgrader and tracks I/O on conn.
func (d *DebugUpgrader) Upgrade(conn io.ReadWriter) (hs ms.Handshake, err error) {
	var (
		// Take the Reader and Writer parts from conn to be probably replaced
		// below.
		r io.Reader = conn
		w io.Writer = conn
	)
	if onRequest := d.OnRequest; onRequest != nil {
		var buf bytes.Buffer
		// First, we must read the entire request.
		req, err := http.ReadRequest(bufio.NewReader(
			io.TeeReader(conn, &buf),
		))
		if err == nil {
			// Fulfill the buffer with the response body.
			io.Copy(ioutil.Discard, req.Body)
			req.Body.Close()
		}
		onRequest(buf.Bytes())

		r = io.MultiReader(
			&buf, conn,
		)
	}

	if onResponse := d.OnResponse; onResponse != nil {
		var buf bytes.Buffer
		// Intercept the response stream written by the Upgrade().
		w = io.MultiWriter(
			conn, &buf,
		)
		defer func() {
			onResponse(buf.Bytes())
		}()
	}

	return d.Upgrader.Upgrade(struct {
		io.Reader
		io.Writer
	}{r, w})
}
