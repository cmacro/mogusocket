package msutil

import (
	"bytes"
	"runtime"
	"testing"

	ms "github.com/cmacro/mogusocket"
)

func TestControlHandler(t *testing.T) {
	for _, test := range []struct {
		name  string
		state ms.State
		in    ms.Frame
		out   ms.Frame
		noOut bool
		err   error
	}{
		{
			name: "ping",
			in:   ms.NewPingFrame(nil),
			out:  ms.NewPongFrame(nil),
		},
		{
			name: "ping",
			in:   ms.NewPingFrame([]byte("catch the ball")),
			out:  ms.NewPongFrame([]byte("catch the ball")),
		},
		{
			name:  "ping",
			state: ms.StateServerSide,
			in:    ms.MaskFrame(ms.NewPingFrame([]byte("catch the ball"))),
			out:   ms.NewPongFrame([]byte("catch the ball")),
		},
		{
			name: "ping",
			in:   ms.NewPingFrame(bytes.Repeat([]byte{0xfe}, 125)),
			out:  ms.NewPongFrame(bytes.Repeat([]byte{0xfe}, 125)),
		},
		{
			name:  "pong",
			in:    ms.NewPongFrame(nil),
			noOut: true,
		},
		{
			name:  "pong",
			in:    ms.NewPongFrame([]byte("caught")),
			noOut: true,
		},
		{
			name: "close",
			in:   ms.NewCloseFrame(nil),
			out:  ms.NewCloseFrame(nil),
			err: ClosedError{
				Code: ms.StatusNoStatusRcvd,
			},
		},
		{
			name: "close",
			in: ms.NewCloseFrame(ms.NewCloseFrameBody(
				ms.StatusGoingAway, "goodbye!",
			)),
			out: ms.NewCloseFrame(ms.NewCloseFrameBody(
				ms.StatusGoingAway, "",
			)),
			err: ClosedError{
				Code:   ms.StatusGoingAway,
				Reason: "goodbye!",
			},
		},
		{
			name: "close",
			in: ms.NewCloseFrame(ms.NewCloseFrameBody(
				ms.StatusGoingAway, "bye",
			)),
			out: ms.NewCloseFrame(ms.NewCloseFrameBody(
				ms.StatusGoingAway, "",
			)),
			err: ClosedError{
				Code:   ms.StatusGoingAway,
				Reason: "bye",
			},
		},
		{
			name:  "close",
			state: ms.StateServerSide,
			in: ms.MaskFrame(ms.NewCloseFrame(ms.NewCloseFrameBody(
				ms.StatusGoingAway, "goodbye!",
			))),
			out: ms.NewCloseFrame(ms.NewCloseFrameBody(
				ms.StatusGoingAway, "",
			)),
			err: ClosedError{
				Code:   ms.StatusGoingAway,
				Reason: "goodbye!",
			},
		},
		{
			name: "close",
			in: ms.NewCloseFrame(ms.NewCloseFrameBody(
				ms.StatusNormalClosure, string([]byte{0, 200}),
			)),
			out: ms.NewCloseFrame(ms.NewCloseFrameBody(
				ms.StatusProtocolError, ms.ErrProtocolInvalidUTF8.Error(),
			)),
			err: ms.ErrProtocolInvalidUTF8,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if err := recover(); err != nil {
					stack := make([]byte, 4096)
					n := runtime.Stack(stack, true)
					t.Fatalf(
						"panic recovered: %v\n%s",
						err, stack[:n],
					)
				}
			}()
			var (
				out = bytes.NewBuffer(nil)
				in  = bytes.NewReader(test.in.Payload)
			)
			c := ControlHandler{
				Src:   in,
				Dst:   out,
				State: test.state,
			}

			err := c.Handle(test.in.Header)
			if err != test.err {
				t.Errorf("unexpected error: %v; want %v", err, test.err)
			}

			if in.Len() != 0 {
				t.Errorf("handler did not drained the input")
			}

			act := out.Bytes()
			switch {
			case len(act) == 0 && test.noOut:
				return
			case len(act) == 0 && !test.noOut:
				t.Errorf("unexpected silence")
			case len(act) > 0 && test.noOut:
				t.Errorf("unexpected sent frame")
			default:
				exp := ms.MustCompileFrame(test.out)
				if !bytes.Equal(act, exp) {
					fa := ms.MustReadFrame(bytes.NewReader(act))
					fe := ms.MustReadFrame(bytes.NewReader(exp))
					t.Errorf(
						"unexpected sent frame:\n\tact: %+v\n\texp: %+v\nbytes:\n\tact: %v\n\texp: %v",
						fa, fe, act, exp,
					)
				}
			}
		})
	}
}
