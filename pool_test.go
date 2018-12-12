// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build all

package gosqlite

import (
	"testing"
	"time"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func TestPool(t *testing.T) {
	pool := NewPool(func() (*Conn, error) {
		return open(t), nil
	}, 3, time.Minute*10)
	for i := 0; i <= 10; i++ {
		c, err := pool.Get()
		checkNoError(t, err, "error getting connection from the pool: %s")
		assert.T(t, c != nil, "expected connection returned by the pool")
		assert.T(t, !c.IsClosed(), "connection returned by the pool is alive")
		_, err = c.SchemaVersion("main")
		checkNoError(t, err, "error using connection from the pool: %s")
		pool.Release(c)
	}
	pool.Close()
	assert.T(t, pool.IsClosed(), "expected pool to be closed")
}

func TestTryGet(t *testing.T) {
	pool := NewPool(func() (*Conn, error) {
		return open(t), nil
	}, 1, time.Minute*10)
	defer pool.Close()
	c, err := pool.TryGet()
	checkNoError(t, err, "error getting connection from the pool: %s")
	assert.T(t, c != nil, "expected connection returned by the pool")
	defer pool.Release(c)

	c1, err := pool.TryGet()
	assert.T(t, c1 == nil && err == nil, "expected no connection returned by the pool")
}
