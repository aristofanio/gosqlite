// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"testing"

	. "github.com/gwenn/gosqlite"
)

func checkCacheSize(t *testing.T, db *Conn, expectedSize, expectedMaxSize int) {
	if size, maxSize := db.CacheSize(); size != expectedSize || maxSize != expectedMaxSize {
		t.Errorf("got size: %d; want %d or got maxsize: %d; want %d", size, expectedSize, maxSize, expectedMaxSize)
	}
}

func TestDisabledCache(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	db.SetCacheSize(0)
	checkCacheSize(t, db, 0, 0)

	s, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "couldn't prepare stmt: %#v")
	if !s.Cacheable {
		t.Error("expected cacheable stmt")
	}

	err = s.Finalize()
	checkNoError(t, err, "couldn't finalize stmt: %#v")
	checkCacheSize(t, db, 0, 0)
}

func TestEnabledCache(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	db.SetCacheSize(10)
	checkCacheSize(t, db, 0, 10)

	s, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "couldn't prepare stmt: %#v")
	if !s.Cacheable {
		t.Error("expected cacheable stmt")
	}

	err = s.Finalize()
	checkNoError(t, err, "couldn't finalize stmt: %#v")
	checkCacheSize(t, db, 1, 10)

	ns, err := db.Prepare("SELECT 1")
	checkNoError(t, err, "couldn't prepare stmt: %#v")
	checkCacheSize(t, db, 0, 10)

	err = ns.Finalize()
	checkNoError(t, err, "couldn't finalize stmt: %#v")
	checkCacheSize(t, db, 1, 10)

	db.SetCacheSize(0)
	checkCacheSize(t, db, 0, 0)
}

func BenchmarkDisabledCache(b *testing.B) {
	db, _ := Open(":memory:")
	defer db.Close()
	db.SetCacheSize(0)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s, _ := db.Prepare("SELECT 1, 'test', 3.14 UNION SELECT 2, 'exp', 2.71")
		s.Finalize()
	}

	if size, maxSize := db.CacheSize(); size != 0 || maxSize != 0 {
		b.Errorf("got size: %d; want %d or got maxsize: %d; want %d", size, 0, maxSize, 0)
	}
}

func BenchmarkEnabledCache(b *testing.B) {
	db, _ := Open(":memory:")
	defer db.Close()
	db.SetCacheSize(10)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		s, _ := db.Prepare("SELECT 1, 'test', 3.14 UNION SELECT 2, 'exp', 2.71")
		s.Finalize()
	}

	if size, maxSize := db.CacheSize(); size != 1 || maxSize != 10 {
		b.Errorf("got size: %d; want %d or got maxsize: %d; want %d", size, 1, maxSize, 10)
	}
}
