package msutil

import (
	"errors"
	"io"
	"strconv"

	ms "github.com/cmacro/mogusocket"
	"github.com/gobwas/pool/pbytes"
)

// ClosedError returned when peer has closed the connection with appropriate
// code and a textual reason.
type ClosedError struct {
	Code   ms.StatusCode
	Reason string
}

// Error implements error interface.
func (err ClosedError) Error() string {
	return "ws closed: " + strconv.FormatUint(uint64(err.Code), 10) + " " + err.Reason
}

// ControlHandler contains logic of handling control frames.
//
// The intentional way to use it is to read the next frame header from the
// connection, optionally check its validity via ms.CheckHeader() and if it is
// not a ms.OpText of ms.OpBinary (or ms.OpContinuation) – pass it to Handle()
// method.
//
// That is, passed header should be checked to get rid of unexpected errors.
//
// The Handle() method will read out all control frame payload (if any) and
// write necessary bytes as a rfc compatible response.
type ControlHandler struct {
	Src   io.Reader
	Dst   io.Writer
	State ms.State

	// DisableSrcCiphering disables unmasking payload data read from Src.
	// It is useful when msutil.Reader is used or when frame payload already
	// pulled and ciphered out from the connection (and introduced by
	// bytes.Reader, for example).
	DisableSrcCiphering bool
}

// ErrNotControlFrame is returned by ControlHandler to indicate that given
// header could not be handled.
var ErrNotControlFrame = errors.New("not a control frame")

// Handle handles control frames regarding to the c.State and writes responses
// to the c.Dst when needed.
//
// It returns ErrNotControlFrame when given header is not of ms.OpClose,
// ms.OpPing or ms.OpPong operation code.
func (c ControlHandler) Handle(h ms.Header) error {
	switch h.OpCode {
	case ms.OpPing:
		return c.HandlePing(h)
	case ms.OpPong:
		return c.HandlePong(h)
	case ms.OpClose:
		return c.HandleClose(h)
	}
	return ErrNotControlFrame
}

// HandlePing handles ping frame and writes specification compatible response
// to the c.Dst.
func (c ControlHandler) HandlePing(h ms.Header) error {
	if h.Length == 0 {
		// The most common case when ping is empty.
		// Note that when sending masked frame the mask for empty payload is
		// just four zero bytes.
		return ms.WriteHeader(c.Dst, ms.Header{
			Fin:    true,
			OpCode: ms.OpPong,
			Masked: c.State.ClientSide(),
		})
	}

	// In other way reply with Pong frame with copied payload.
	p := pbytes.GetLen(int(h.Length) + ms.HeaderSize(ms.Header{
		Length: h.Length,
		Masked: c.State.ClientSide(),
	}))
	defer pbytes.Put(p)

	// Deal with ciphering i/o:
	// Masking key is used to mask the "Payload data" defined in the same
	// section as frame-payload-data, which includes "Extension data" and
	// "Application data".
	//
	// See https://tools.ietf.org/html/rfc6455#section-5.3
	//
	// NOTE: We prefer ControlWriter with preallocated buffer to
	// ms.WriteHeader because it performs one syscall instead of two.
	w := NewControlWriterBuffer(c.Dst, c.State, ms.OpPong, p)
	r := c.Src
	if c.State.ServerSide() && !c.DisableSrcCiphering {
		r = NewCipherReader(r, h.Mask)
	}

	_, err := io.Copy(w, r)
	if err == nil {
		err = w.Flush()
	}

	return err
}

// HandlePong handles pong frame by discarding it.
func (c ControlHandler) HandlePong(h ms.Header) error {
	if h.Length == 0 {
		return nil
	}

	buf := pbytes.GetLen(int(h.Length))
	defer pbytes.Put(buf)

	// Discard pong message according to the RFC6455:
	// A Pong frame MAY be sent unsolicited. This serves as a
	// unidirectional heartbeat. A response to an unsolicited Pong frame
	// is not expected.
	_, err := io.CopyBuffer(io.Discard, c.Src, buf)

	return err
}

// HandleClose handles close frame, makes protocol validity checks and writes
// specification compatible response to the c.Dst.
func (c ControlHandler) HandleClose(h ms.Header) error {
	if h.Length == 0 {
		err := ms.WriteHeader(c.Dst, ms.Header{
			Fin:    true,
			OpCode: ms.OpClose,
			Masked: c.State.ClientSide(),
		})
		if err != nil {
			return err
		}

		// Due to RFC, we should interpret the code as no status code
		// received:
		//   If this Close control frame contains no status code, _The WebSocket
		//   Connection Close Code_ is considered to be 1005.
		//
		// See https://tools.ietf.org/html/rfc6455#section-7.1.5
		return ClosedError{
			Code: ms.StatusNoStatusRcvd,
		}
	}

	// Prepare bytes both for reading reason and sending response.
	p := pbytes.GetLen(int(h.Length) + ms.HeaderSize(ms.Header{
		Length: h.Length,
		Masked: c.State.ClientSide(),
	}))
	defer pbytes.Put(p)

	// Get the subslice to read the frame payload out.
	subp := p[:h.Length]

	r := c.Src
	if c.State.ServerSide() && !c.DisableSrcCiphering {
		r = NewCipherReader(r, h.Mask)
	}
	if _, err := io.ReadFull(r, subp); err != nil {
		return err
	}

	code, reason := ms.ParseCloseFrameData(subp)
	if err := ms.CheckCloseFrameData(code, reason); err != nil {
		// Here we could not use the prepared bytes because there is no
		// guarantee that it may fit our protocol error closure code and a
		// reason.
		c.closeWithProtocolError(err)
		return err
	}

	// Deal with ciphering i/o:
	// Masking key is used to mask the "Payload data" defined in the same
	// section as frame-payload-data, which includes "Extension data" and
	// "Application data".
	//
	// See https://tools.ietf.org/html/rfc6455#section-5.3
	//
	// NOTE: We prefer ControlWriter with preallocated buffer to
	// ms.WriteHeader because it performs one syscall instead of two.
	w := NewControlWriterBuffer(c.Dst, c.State, ms.OpClose, p)

	// RFC6455#5.5.1:
	// If an endpoint receives a Close frame and did not previously
	// send a Close frame, the endpoint MUST send a Close frame in
	// response. (When sending a Close frame in response, the endpoint
	// typically echoes the status code it received.)
	_, err := w.Write(p[:2])
	if err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return ClosedError{
		Code:   code,
		Reason: reason,
	}
}

func (c ControlHandler) closeWithProtocolError(reason error) error {
	f := ms.NewCloseFrame(ms.NewCloseFrameBody(
		ms.StatusProtocolError, reason.Error(),
	))
	if c.State.ClientSide() {
		ms.MaskFrameInPlace(f)
	}
	return ms.WriteFrame(c.Dst, f)
}
