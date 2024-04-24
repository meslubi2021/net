// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package html

import (
	"testing"
)

func init() {
	UnescapeString("") // force load of entity maps
}

func TestEntityLength(t *testing.T) {
	if len(entity) == 0 {
		t.Fatal("maps not loaded")
	}

	// We verify that the length of UTF-8 encoding of each value
	// is no more than 1 + len("&"+key), which is an assuption
	// made in unescapeEntity.
	for k, v := range entity {
		if 2+len(k) < int(v[0]) {
			t.Error("escaped entity &" + k + " is more than 1 byte shorter than its UTF-8 encoding " + string(v[1:1+v[0]]))
		}
		if len(k) > longestEntityWithoutSemicolon && k[len(k)-1] != ';' {
			t.Errorf("entity name %s is %d characters, but longestEntityWithoutSemicolon=%d", k, len(k), longestEntityWithoutSemicolon)
		}
	}
}
