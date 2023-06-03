package main

import (
	"context"
	"io"
	"strconv"
	"sync"

	ms "github.com/cmacro/mogusocket"
)

func NewTestSections(log ms.Logger) *Sections {
	return &Sections{
		log:   log,
		items: make(map[int64]*Client),
		Mutex: &sync.Mutex{},
	}
}

type Client struct {
	id     int64
	log    ms.Logger
	writer ms.SendFunc
	ctx    context.Context
	cancel func()
}

type Sections struct {
	log ms.Logger
	*sync.Mutex

	maxid int64
	items map[int64]*Client

	// Connect(ctx context.Context, w func(src io.Reader, isText bool) error, c func()) int64
	// Close(id int64)
	// ReadDump(r io.Reader) error
}

func (s *Sections) Connect(ctx context.Context, w ms.SendFunc, c func()) (ms.SectionHandler, error) {
	s.Lock()
	s.maxid++
	nid := s.maxid
	section := &Client{id: nid, writer: w, ctx: ctx, cancel: c, log: s.log.Sub(strconv.FormatInt(nid, 10))}
	s.items[nid] = section
	s.Unlock()

	return section, nil
}

func (s *Sections) Close(section ms.SectionHandler) error {
	// remove
	id := section.GetId()
	s.Lock()
	delete(s.items, id)
	s.Unlock()
	section.Close()
	return nil
}

// func (s *TestSections) ReadDump(r io.Reader, isText bool) error {
// 	s.Log.Info("read dump", isText)
// 	err := s.writer(r, isText)
// 	return err
// }

func (c *Client) Close() {
	c.cancel()
}

func (c *Client) ReadDump(r io.Reader, isText bool) error {
	c.log.Info("read dump", isText)
	err := c.writer(r, isText)
	return err
}

func (c *Client) GetId() int64 {
	return c.id
}
