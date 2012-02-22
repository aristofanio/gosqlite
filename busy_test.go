package sqlite_test

import (
	. "github.com/gwenn/gosqlite"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

/*
func TestInterrupt(t *testing.T) {
	db := open(t)
	defer db.Close()
	db.CreateScalarFunction("interrupt", 0, nil, func(ctx *Context, nArg int) {
		//db.Interrupt()
		ctx.ResultText("ok")
		}, nil)
	var err error
	var result string
	err = db.OneValue("select interrupt()", &result)
	if err == nil || err != ErrInterrupt {
		t.Errorf("Expected interrupt but got %#v", err)
	}
}
*/

func openTwoConnSameDb(t *testing.T) (*os.File, *Conn, *Conn) {
	f, err := ioutil.TempFile("", "gosqlite-test")
	checkNoError(t, f.Close(), "couldn't close temp file: %s")
	db1, err := Open(f.Name(), OPEN_READWRITE, OPEN_CREATE, OPEN_FULLMUTEX)
	checkNoError(t, err, "couldn't open database file: %s")
	db2, err := Open(f.Name(), OPEN_READWRITE, OPEN_CREATE, OPEN_FULLMUTEX)
	checkNoError(t, err, "couldn't open database file: %s")
	return f, db1, db2
}

func TestDefaultBusy(t *testing.T) {
	f, db1, db2 := openTwoConnSameDb(t)
	defer os.Remove(f.Name())
	defer db1.Close()
	defer db2.Close()
	checkNoError(t, db1.BeginTransaction(EXCLUSIVE), "couldn't begin transaction: %s")
	defer db1.Rollback()

	_, err := db2.SchemaVersion()
	if err == nil {
		t.Fatalf("Expected lock but got %v", err)
	}
	if se, ok := err.(*StmtError); !ok || se.Code() != ErrBusy {
		t.Fatalf("Exepted lock but got %#v", err)
	}
}

func TestBusyTimeout(t *testing.T) {
	f, db1, db2 := openTwoConnSameDb(t)
	defer os.Remove(f.Name())
	defer db1.Close()
	defer db2.Close()
	checkNoError(t, db1.BeginTransaction(EXCLUSIVE), "couldn't begin transaction: %s")

	//join := make(chan bool)
	checkNoError(t, db2.BusyTimeout(500), "couldn't set busy timeout: %s")
	go func() {
		time.Sleep(time.Millisecond)
		db1.Rollback()
		//join <- true
	}()

	_, err := db2.SchemaVersion()
	checkNoError(t, err, "couldn't query schema version: %#v")
	//<- join
}

func TestBusyHandler(t *testing.T) {
	f, db1, db2 := openTwoConnSameDb(t)
	defer os.Remove(f.Name())
	defer db1.Close()
	defer db2.Close()

	//c := make(chan bool)
	var called bool
	err := db2.BusyHandler(func(udp interface{}, count int) bool {
		if b, ok := udp.(*bool); ok {
			*b = true
		}
		//c <- true
		return true
	}, &called)

	checkNoError(t, db1.BeginTransaction(EXCLUSIVE), "couldn't begin transaction: %s")

	go func() {
		time.Sleep(time.Millisecond)
		//_ = <- c
		db1.Rollback()
	}()

	_, err = db2.SchemaVersion()
	checkNoError(t, err, "couldn't query schema version: %#v")
	if !called {
		t.Fatalf("Busy handler not called!")
	}
}
