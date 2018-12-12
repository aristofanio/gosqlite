// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func TestIntegrityCheck(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	checkNoError(t, db.IntegrityCheck("", 1, true), "Error checking integrity of database: %s")
	checkNoError(t, db.IntegrityCheck("", 1, false), "Error checking integrity of database: %s")
	err := db.IntegrityCheck("bim", 1, true)
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestEncoding(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	encoding, err := db.Encoding("")
	checkNoError(t, err, "Error reading encoding of database: %s")
	assert.Equal(t, "UTF-8", encoding)

	_, err = db.Encoding("bim")
	assert.T(t, err != nil)
}

func TestSchemaVersion(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	version, err := db.SchemaVersion("")
	checkNoError(t, err, "Error reading schema version of database: %s")
	assert.Equal(t, 0, version)
}

func TestJournalMode(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	mode, err := db.JournalMode("")
	checkNoError(t, err, "Error reading journaling mode of database: %s")
	assert.Equal(t, "memory", mode)

	_, err = db.JournalMode("bim")
	assert.T(t, err != nil)
}

func TestSetJournalMode(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	mode, err := db.SetJournalMode("", "OFF")
	checkNoError(t, err, "Error setting journaling mode of database: %s")
	assert.Equal(t, "off", mode)

	_, err = db.SetJournalMode("bim", "OFF")
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestLockingMode(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	mode, err := db.LockingMode("")
	checkNoError(t, err, "Error reading locking-mode of database: %s")
	assert.Equal(t, "normal", mode)

	_, err = db.LockingMode("bim")
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestSetLockingMode(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	mode, err := db.SetLockingMode("", "exclusive")
	checkNoError(t, err, "Error setting locking-mode of database: %s")
	assert.Equal(t, "exclusive", mode)

	_, err = db.SetLockingMode("bim", "exclusive")
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestSynchronous(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	mode, err := db.Synchronous("")
	checkNoError(t, err, "Error reading synchronous flag of database: %s")
	assert.Equal(t, 2, mode)

	_, err = db.Synchronous("bim")
	assert.T(t, err != nil)
}

func TestSetSynchronous(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.SetSynchronous("", 0)
	checkNoError(t, err, "Error setting synchronous flag of database: %s")
	mode, err := db.Synchronous("")
	checkNoError(t, err, "Error reading synchronous flag of database: %s")
	assert.Equal(t, 0, mode)
}

func TestQueryOnly(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	mode, err := db.QueryOnly("")
	if err == io.EOF {
		return // not supported
	}
	checkNoError(t, err, "Error reading query_only status of database: %s")
	assert.T(t, !mode, "expecting query_only to be false by default")
	err = db.SetQueryOnly("", true)
	checkNoError(t, err, "Error setting query_only status of database: %s")
	err = db.Exec("CREATE TABLE test (data TEXT)")
	assert.T(t, err != nil, "expected error")
	//println(err.Error())

	_, err = db.QueryOnly("bim")
	assert.T(t, err != nil)
}

func TestApplicationID(t *testing.T) {
	if VersionNumber() < 3007017 {
		t.Skipf("SQLite version too old (%d < %d)", VersionNumber(), 3007017)
	}

	db := open(t)
	defer checkClose(db, t)

	appID, err := db.ApplicationID("")
	checkNoError(t, err, "error getting application Id: %s")
	assert.Equalf(t, 0, appID, "got: %d; want: %d", appID, 0)

	err = db.SetApplicationID("", 123)
	checkNoError(t, err, "error setting application Id: %s")

	appID, err = db.ApplicationID("")
	checkNoError(t, err, "error getting application Id: %s")
	assert.Equalf(t, 123, appID, "got: %d; want: %d", appID, 123)

	_, err = db.ApplicationID("bim")
	assert.T(t, err != nil)
}

func TestForeignKeyCheck(t *testing.T) {
	if VersionNumber() < 3007016 {
		t.Skipf("SQLite version too old (%d < %d)", VersionNumber(), 3007016)
	}

	db := open(t)
	defer checkClose(db, t)
	checkNoError(t, db.Exec(`
		CREATE TABLE tree (
		id INTEGER PRIMARY KEY NOT NULL,
		parentId INTEGER,
		name TEXT NOT NULL,
		FOREIGN KEY (parentId) REFERENCES tree(id)
		);
	  INSERT INTO tree VALUES (0, NULL, 'root'),
	  (1, 0, 'node1'),
	  (2, 0, 'node2'),
	  (3, 1, 'leaf'),
	  (4, 5, 'orphan')
	  ;
	`), "%s")
	vs, err := db.ForeignKeyCheck("", "tree")
	checkNoError(t, err, "error while checking FK: %s")
	assert.Equal(t, 1, len(vs), "one FK violation expected")
	v := vs[0]
	assert.Equal(t, FkViolation{Table: "tree", RowID: 4, Parent: "tree", FkID: 0}, v)
	fks, err := db.ForeignKeys("", "tree")
	checkNoError(t, err, "error while loading FK: %s")
	fk, ok := fks[v.FkID]
	assert.Tf(t, ok, "no FK with id: %d", v.FkID)
	assert.Equal(t, &ForeignKey{Table: "tree", From: []string{"parentId"}, To: []string{"id"}}, fk)

	mvs, err := db.ForeignKeyCheck("main", "tree")
	checkNoError(t, err, "error while checking FK: %s")
	assert.Equal(t, vs, mvs)

	mvs, err = db.ForeignKeyCheck("main", "")
	checkNoError(t, err, "error while checking FK: %s")
	assert.Equal(t, vs, mvs)

	mvs, err = db.ForeignKeyCheck("", "")
	checkNoError(t, err, "error while checking FK: %s")
	assert.Equal(t, vs, mvs)
}

func TestMMapSize(t *testing.T) {
	if VersionNumber() < 3007017 {
		t.Skipf("SQLite version too old (%d < %d)", VersionNumber(), 3007017)
	}
	f, err := ioutil.TempFile("", "gosqlite.db.")
	checkNoError(t, err, "couldn't create temp file: %s")
	checkNoError(t, f.Close(), "couldn't close temp file: %s")
	defer os.Remove(f.Name())

	db, err := Open(f.Name(), OpenReadWrite, OpenCreate, OpenFullMutex)
	checkNoError(t, err, "couldn't open database file: %s")

	size, err := db.MMapSize("")
	checkNoError(t, err, "error while getting mmap size: %s")
	assert.Equal(t, int64(0), size)

	newSize, err := db.SetMMapSize("", 1048576)
	checkNoError(t, err, "error while setting mmap size: %s")
	assert.Equal(t, int64(1048576), newSize)
}
