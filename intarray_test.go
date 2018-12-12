// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"

	"testing"
)

func TestIntArrayModule(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	err := db.FastExec(`CREATE TABLE t1 (x INT);
		INSERT INTO t1 VALUES (1), (3);
		CREATE TABLE t2 (y INT);
		INSERT INTO t2 VALUES (11);
		CREATE TABLE t3 (z INT);
		INSERT INTO t3 VALUES (-5);`)
	assert.T(t, err == nil)

	var p1, p2, p3 IntArray
	p1, err = db.CreateIntArray("ex1")
	assert.T(t, err == nil)
	p2, err = db.CreateIntArray("ex2")
	assert.T(t, err == nil)
	p3, err = db.CreateIntArray("ex3")
	assert.T(t, err == nil)

	s, err := db.Prepare(`SELECT * FROM t1, t2, t3
	 WHERE t1.x IN ex1
	  AND t2.y IN ex2
	  AND t3.z IN ex3`)
	assert.T(t, err == nil)
	defer checkFinalize(s, t)

	p1.Bind([]int64{1, 2, 3, 4})
	p2.Bind([]int64{5, 6, 7, 8, 9, 10, 11})
	// Fill in content of a3
	p3.Bind([]int64{-1, -5, -10})

	var i1, i2, i3 int
	for checkStep(t, s) {
		err = s.Scan(&i1, &i2, &i3)
		assert.T(t, err == nil)
		assert.T(t, i1 == 1 || i1 == 3)
		assert.Equal(t, 11, i2)
		assert.Equal(t, -5, i3)
	}

	s.Reset()
	p1.Bind([]int64{1})
	p2.Bind([]int64{7, 11})
	p3.Bind([]int64{-5, -10})
	assert.T(t, checkStep(t, s))
	err = s.Scan(&i1, &i2, &i3)
	assert.T(t, err == nil)
	assert.Equal(t, 1, i1)
	assert.Equal(t, 11, i2)
	assert.Equal(t, -5, i3)

	s.Reset()
	p1.Bind([]int64{1})
	p2.Bind([]int64{3, 4, 5})
	p3.Bind([]int64{0, -5})
	assert.T(t, !checkStep(t, s))

	checkNoError(t, p1.Drop(), "%s")
	checkNoError(t, p2.Drop(), "%s")
	checkNoError(t, p3.Drop(), "%s")
}

const IntArraySize = 100

func BenchmarkNoIntArray(b *testing.B) {
	b.StopTimer()
	db, err := Open(":memory:")
	panicOnError(b, err)
	defer db.Close()

	panicOnError(b, db.Exec("CREATE TABLE rand (r INT)"))
	values := rand.Perm(IntArraySize)
	for _, v := range values {
		panicOnError(b, db.Exec("INSERT INTO rand (r) VALUES (?)", v))
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		l := rand.Intn(IntArraySize-1) + 1 // at least one value
		sql := fmt.Sprintf("SELECT * FROM rand WHERE r IN (?%s)", strings.Repeat(", ?", l-1))
		s, err := db.Prepare(sql)
		panicOnError(b, err)
		for j := 0; j < l; j++ {
			panicOnError(b, s.BindByIndex(j+1, values[j]))
		}
		nr := 0
		err = s.Select(func(s *Stmt) error {
			_, _, err = s.ScanInt32(0)
			nr++
			return err
		})
		panicOnError(b, err)
		if nr != l {
			b.Fatalf("Only %d values; %d expected", nr, l)
		}
		panicOnError(b, s.Finalize())
	}
}

func BenchmarkIntArray(b *testing.B) {
	b.StopTimer()
	db, err := Open(":memory:")
	panicOnError(b, err)
	defer db.Close()

	panicOnError(b, db.Exec("CREATE TABLE rand (r INT)"))
	perms := rand.Perm(IntArraySize)
	values := make([]int64, len(perms))
	for i, v := range perms {
		panicOnError(b, db.Exec("INSERT INTO rand (r) VALUES (?)", v))
		values[i] = int64(v)
	}

	b.StartTimer()

	p, err := db.CreateIntArray("myia")
	panicOnError(b, err)
	defer p.Drop()
	s, err := db.Prepare("SELECT * FROM rand WHERE r IN myia")
	panicOnError(b, err)
	defer s.Finalize()

	for i := 0; i < b.N; i++ {
		l := rand.Intn(IntArraySize-1) + 1 // at least one value
		p.Bind(values[0:l])
		nr := 0
		err = s.Select(func(s *Stmt) error {
			_, _, err = s.ScanInt32(0)
			nr++
			return err
		})
		panicOnError(b, err)
		if nr != l {
			b.Fatalf("Only %d values; %d expected", nr, l)
		}
	}
}
