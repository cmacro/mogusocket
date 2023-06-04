package mogusocket

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParserAddr(t *testing.T) {
	var u *Addr
	var err error
	u, err = ParserAddr("unix:///tmp.socket")
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, u.Network, "unix")
	assert.Equal(t, u.Address, "/tmp.socket")
	u, err = ParserAddr("tcp://localhost:9001")
	assert.Nil(t, err)
	assert.Equal(t, u.Network, "tcp")
	assert.Equal(t, u.Address, "localhost:9001")

	u, err = ParserAddr("tcp://golang.org")
	assert.Nil(t, err)
	assert.Equal(t, u.Network, "tcp")
	assert.Equal(t, u.Address, "golang.org")

	// u, err = ParserAddr("udp://[2001:db8::1]:domain")
	// assert.Nil(t, err)
	// assert.Equal(t, u.Network, "udp")
	// assert.Equal(t, u.Address, "[2001:db8::1]:domain")

	u, err = ParserAddr("udp://:80")
	assert.Nil(t, err)
	assert.Equal(t, u.Network, "udp")
	assert.Equal(t, u.Address, ":80")

	u, err = ParserAddr("ip4://192.0.2.1")
	assert.Nil(t, err)
	assert.Equal(t, u.Network, "ip4")
	assert.Equal(t, u.Address, "192.0.2.1")

	u, err = ParserAddr("ip6://2001:db8::1")
	assert.Nil(t, err)
	assert.Equal(t, u.Network, "ip6")
	assert.Equal(t, u.Address, "2001:db8::1")

	// u, err = ParserAddr("ip6://fe80::1%lo0")
	// assert.Nil(t, err)
	// assert.Equal(t, u.Network, "ip6")
	// assert.Equal(t, u.Address, "fe80::1%lo0")

	// 	Dial("ip4:1", "192.0.2.1")
	// 	Dial("ip6:ipv6-icmp", "2001:db8::1")
	// 	Dial("ip6:58", "fe80::1%lo0")

}
