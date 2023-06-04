//go:build go1.8
// +build go1.8

package mogusocket

import "crypto/tls"

func tlsCloneConfig(c *tls.Config) *tls.Config {
	return c.Clone()
}
