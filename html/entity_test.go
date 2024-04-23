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

	for k := range entity {
		if len(k) > longestEntityWithoutSemicolon && k[len(k)-1] != ';' {
			t.Errorf("entity name %s is %d characters, but longestEntityWithoutSemicolon=%d", k, len(k), longestEntityWithoutSemicolon)
		}
	}
}
