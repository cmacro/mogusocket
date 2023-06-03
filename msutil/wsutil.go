/*
package msutil provides utilities for working with WebSocket protocol.

Overview:

	// Read masked text message from peer and check utf8 encoding.
	header, err := ws.ReadHeader(conn)
	if err != nil {
		// handle err
	}

	// Prepare to read payload.
	r := io.LimitReader(conn, header.Length)
	r = msutil.NewCipherReader(r, header.Mask)
	r = msutil.NewUTF8Reader(r)

	payload, err := io.ReadAll(r)
	if err != nil {
		// handle err
	}

You could get the same behavior using just `msutil.Reader`:

	r := msutil.Reader{
		Source:    conn,
		CheckUTF8: true,
	}

	payload, err := io.ReadAll(r)
	if err != nil {
		// handle err
	}

Or even simplest:

	payload, err := msutil.ReadClientText(conn)
	if err != nil {
		// handle err
	}

Package is also exports tools for buffered writing:

	// Create buffered writer, that will buffer output bytes and send them as
	// 128-length fragments (with exception on large writes, see the doc).
	writer := msutil.NewWriterSize(conn, ws.StateServerSide, ws.OpText, 128)

	_, err := io.CopyN(writer, rand.Reader, 100)
	if err == nil {
		err = writer.Flush()
	}
	if err != nil {
		// handle error
	}

For more utils and helpers see the documentation.
*/
package msutil
