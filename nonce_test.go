// Copyright 2023 @moguf.com All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file

package mogusocket

import "testing"

func BenchmarkInitAcceptFromNonce(b *testing.B) {
	dst := make([]byte, acceptSize)
	nonce := mustMakeNonce()
	for i := 0; i < b.N; i++ {
		initAcceptFromNonce(dst, nonce)
	}
}
