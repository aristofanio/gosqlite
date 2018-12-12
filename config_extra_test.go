// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build all

package gosqlite

import (
	"fmt"
	"testing"

	. "github.com/gwenn/gosqlite"
)

func init() {
	err := EnableSharedCache(false)
	if err != nil {
		panic(fmt.Sprintf("couldn't disable shared cache: '%s'", err))
	}
	if VersionNumber() >= 3007017 {
		err = ConfigMMapSize(0, 268435456)
		if err != nil {
			panic(fmt.Sprintf("cannot change mmap size: '%s'", err))
		}
	}
}

func TestEnableLoadExtension(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.EnableLoadExtension(false)
	checkNoError(t, err, "EnableLoadExtension error: %s")
}
