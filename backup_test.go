// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"testing"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func TestBackup(t *testing.T) {
	dst := open(t)
	defer checkClose(dst, t)
	src := open(t)
	defer checkClose(src, t)
	fill(nil, src, 1000)

	bck, err := NewBackup(dst, "main", src, "main")
	checkNoError(t, err, "couldn't init backup: %#v")

	cbs := make(chan BackupStatus)
	defer close(cbs)
	go func() {
		for {
			s := <-cbs
			t.Logf("Backup progress %#v\n", s)
			if s.Remaining == 0 {
				break
			}
		}
	}()
	err = bck.Run(10, 0, cbs)
	checkNoError(t, err, "couldn't do backup: %#v")

	err = bck.Close()
	checkNoError(t, err, "couldn't close backup twice: %#v")
}

func TestBackupMisuse(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	bck, err := NewBackup(db, "", db, "")
	assert.T(t, bck == nil && err != nil, "source and destination must be distinct")
	err = bck.Run(10, 0, nil)
	assert.T(t, err != nil, "misuse expected")
	//println(err.Error())

	bck, err = NewBackup(db, "", nil, "")
	assert.T(t, err != nil, "error expected")
	//println(err.Error())
}
