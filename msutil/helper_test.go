package msutil

import (
	"bytes"
	"io"
	"testing"

	ms "github.com/cmacro/mogusocket"
)

func TestReadMessageEOF(t *testing.T) {
	for _, test := range []struct {
		source   func() io.Reader
		messages []Message
		err      error
	}{
		{
			source: func() io.Reader { return eofReader },
			err:    io.EOF,
		},
		{
			source: func() io.Reader {
				// This case tests that ReadMessage still fails after
				// successfully reading header bytes frame via ms.ReadHeader()
				// and non-successfully read of the body.
				var buf bytes.Buffer
				f := ms.NewTextFrame([]byte("this part will be lost"))
				if err := ms.WriteHeader(&buf, f.Header); err != nil {
					panic(err)
				}
				return &buf
			},
			err: io.ErrUnexpectedEOF,
		},
		{
			source: func() io.Reader {
				// This case tests that ReadMessage not fail when reading
				// fragmented messages.
				var buf bytes.Buffer
				fs := []ms.Frame{
					ms.NewFrame(ms.OpText, false, []byte("fragment1")),
					ms.NewFrame(ms.OpContinuation, false, []byte(",")),
					ms.NewFrame(ms.OpContinuation, true, []byte("fragment2")),
				}
				for _, f := range fs {
					if err := ms.WriteFrame(&buf, f); err != nil {
						panic(err)
					}
				}
				return &buf
			},
			messages: []Message{
				{ms.OpText, []byte("fragment1,fragment2")},
			},
		},
	} {
		t.Run("", func(t *testing.T) {
			ms, err := ReadMessage(test.source(), 0, nil)
			if err != test.err {
				t.Errorf("unexpected error: %v; want %v", err, test.err)
			}
			if n := len(ms); n != len(test.messages) {
				t.Fatalf("unexpected number of read messages: %d; want %d", n, 0)
			}
			for i, exp := range test.messages {
				act := ms[i]
				if act.OpCode != exp.OpCode {
					t.Errorf(
						"unexpected #%d message op code: %v; want %v",
						i, act.OpCode, exp.OpCode,
					)
				}
				if !bytes.Equal(act.Payload, exp.Payload) {
					t.Errorf(
						"unexpected #%d message payload: %q; want %q",
						i, string(act.Payload), string(exp.Payload),
					)
				}
			}
		})
	}
}
