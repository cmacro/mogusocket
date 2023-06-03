package mogusocket

import (
	"context"
	"sync"
)

func NewClient(addr string, log Logger) *Client {
	return &Client{
		addr: addr,
		log:  log,
	}
}

type Client struct {
	addr string
	log  Logger
	mu   sync.Mutex
	// conn net.Conn
}

func (c *Client) Run(ctx context.Context) {
	// u, err := ParserAddr(c.addr)
	// if err != nil {
	// 	c.log.Error("parser addr", err)
	// 	return
	// }
	// conn, err := net.Dial(u.Data())
	// if err != nil {
	// 	c.log.Error("client dial connect", err)
	// 	return
	// }

	// go func() {
	// 	conn.Read()
	// }()

}
