// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func TestInterrupt(t *testing.T) {
	skipIfCgoCheckActive(t)

	db := open(t)
	defer checkClose(db, t)
	db.CreateScalarFunction("interrupt", 0, false, nil, func(ctx *ScalarContext, nArg int) {
		db.Interrupt()
		ctx.ResultText("ok")
	}, nil)
	s, err := db.Prepare("SELECT interrupt() FROM (SELECT 1 UNION SELECT 2 UNION SELECT 3)")
	checkNoError(t, err, "couldn't prepare stmt: %#v")
	defer checkFinalize(s, t)
	err = s.Select(func(s *Stmt) (err error) {
		return
	})
	if err == nil {
		t.Fatalf("got %v; want interrupt", err)
	}
	//println(err.Error())
	if se, ok := err.(StmtError); !ok || se.Code() != ErrInterrupt {
		t.Errorf("got %#v; want interrupt", err)
	}
}

func openTwoConnSameDb(t *testing.T) (*os.File, *Conn, *Conn) {
	f, err := ioutil.TempFile("", "gosqlite-test")
	checkNoError(t, err, "couldn't create temp file: %s")
	checkNoError(t, f.Close(), "couldn't close temp file: %s")
	db1, err := Open(f.Name(), OpenReadWrite, OpenCreate, OpenFullMutex)
	checkNoError(t, err, "couldn't open database file: %s")
	db2, err := Open(f.Name(), OpenReadWrite, OpenCreate, OpenFullMutex)
	checkNoError(t, err, "couldn't open database file: %s")
	return f, db1, db2
}

func TestDefaultBusy(t *testing.T) {
	f, db1, db2 := openTwoConnSameDb(t)
	defer os.Remove(f.Name())
	defer checkClose(db1, t)
	defer checkClose(db2, t)
	checkNoError(t, db1.BeginTransaction(Exclusive), "couldn't begin transaction: %s")
	defer db1.Rollback()

	_, err := db2.SchemaVersion("")
	if err == nil {
		t.Fatalf("got %v; want lock", err)
	}
	if se, ok := err.(StmtError); !ok || se.Code() != ErrBusy {
		t.Fatalf("got %#v; want lock", err)
	}
}

func TestBusyTimeout(t *testing.T) {
	f, db1, db2 := openTwoConnSameDb(t)
	defer os.Remove(f.Name())
	defer checkClose(db1, t)
	defer checkClose(db2, t)
	checkNoError(t, db1.BeginTransaction(Exclusive), "couldn't begin transaction: %s")

	//join := make(chan bool)
	checkNoError(t, db2.BusyTimeout(500*time.Millisecond), "couldn't set busy timeout: %s")
	go func() {
		time.Sleep(time.Millisecond)
		db1.Rollback()
		//join <- true
	}()

	_, err := db2.SchemaVersion("")
	checkNoError(t, err, "couldn't query schema version: %#v")
	//<- join
}

func TestBusyHandler(t *testing.T) {
	skipIfCgoCheckActive(t)

	f, db1, db2 := openTwoConnSameDb(t)
	defer os.Remove(f.Name())
	defer checkClose(db1, t)
	defer checkClose(db2, t)

	//c := make(chan bool)
	var called bool
	err := db2.BusyHandler(func(udp interface{}, count int) bool {
		if b, ok := udp.(*bool); ok {
			*b = true
		}
		//c <- true
		time.Sleep(time.Millisecond)
		return true
	}, &called)

	checkNoError(t, db1.BeginTransaction(Exclusive), "couldn't begin transaction: %s")

	go func() {
		time.Sleep(time.Millisecond)
		//_ = <- c
		db1.Rollback()
	}()

	_, err = db2.SchemaVersion("")
	checkNoError(t, err, "couldn't query schema version: %#v")
	assert.T(t, called, "expected busy handler to be called")
}
