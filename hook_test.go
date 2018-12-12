// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"fmt"
	"testing"

	. "github.com/gwenn/gosqlite"
)

func commitHook(d interface{}) bool {
	if t, ok := d.(*testing.T); ok {
		t.Log("CMT")
	} else {
		fmt.Println(d)
	}
	return false
}

func rollbackHook(d interface{}) {
	if t, ok := d.(*testing.T); ok {
		t.Log("RBK")
	} else {
		fmt.Println(d)
	}
}

func updateHook(d interface{}, a Action, dbName, tableName string, rowID int64) {
	if t, ok := d.(*testing.T); ok {
		t.Logf("UPD: %d, %s.%s.%d\n", a, dbName, tableName, rowID)
	} else {
		fmt.Printf("%s: %d, %s.%s.%d\n", d, a, dbName, tableName, rowID)
	}
}

func TestNoHook(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	db.CommitHook(nil, nil)
	db.RollbackHook(nil, nil)
	db.UpdateHook(nil, nil)
}

func TestCommitHook(t *testing.T) {
	skipIfCgoCheckActive(t)

	db := open(t)
	defer checkClose(db, t)

	db.CommitHook(commitHook, t)
	checkNoError(t, db.Begin(), "%s")
	createTable(db, t)
	checkNoError(t, db.Commit(), "%s")
}

func TestRollbackHook(t *testing.T) {
	skipIfCgoCheckActive(t)

	db := open(t)
	defer checkClose(db, t)

	db.RollbackHook(rollbackHook, t)
	checkNoError(t, db.Begin(), "%s")
	createTable(db, t)
	checkNoError(t, db.Rollback(), "%s")
}

func TestUpdateHook(t *testing.T) {
	skipIfCgoCheckActive(t)

	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)

	db.UpdateHook(updateHook, t)
	checkNoError(t, db.Exec("INSERT INTO test VALUES (1, 273.1, 0, 'data')"), "%s")
}
