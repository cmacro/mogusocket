package msutil

import ms "github.com/cmacro/mogusocket"

// RecvExtension is an interface for clearing fragment header RSV bits.
type RecvExtension interface {
	UnsetBits(ms.Header) (ms.Header, error)
}

// RecvExtensionFunc is an adapter to allow the use of ordinary functions as
// RecvExtension.
type RecvExtensionFunc func(ms.Header) (ms.Header, error)

// BitsRecv implements RecvExtension.
func (fn RecvExtensionFunc) UnsetBits(h ms.Header) (ms.Header, error) {
	return fn(h)
}

// SendExtension is an interface for setting fragment header RSV bits.
type SendExtension interface {
	SetBits(ms.Header) (ms.Header, error)
}

// SendExtensionFunc is an adapter to allow the use of ordinary functions as
// SendExtension.
type SendExtensionFunc func(ms.Header) (ms.Header, error)

// BitsSend implements SendExtension.
func (fn SendExtensionFunc) SetBits(h ms.Header) (ms.Header, error) {
	return fn(h)
}
