// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"testing"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func createIndex(db *Conn, t *testing.T) {
	err := db.Exec("DROP INDEX IF EXISTS test_index;" +
		"CREATE INDEX test_index on test(a_string)")
	checkNoError(t, err, "error creating index: %s")
}

func TestDatabases(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	databases, err := db.Databases()
	checkNoError(t, err, "error looking for databases: %s")
	if len(databases) != 1 {
		t.Errorf("got %d database(s); want one", len(databases))
	}
	if _, ok := databases["main"]; !ok {
		t.Errorf("Expected 'main' database\n")
	}
}

func TestTables(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	tables, err := db.Tables("")
	checkNoError(t, err, "error looking for tables: %s")
	assert.Equal(t, 0, len(tables), "table count")
	createTable(db, t)
	tables, err = db.Tables("main")
	checkNoError(t, err, "error looking for tables: %s")
	assert.Equal(t, 1, len(tables), "table count")
	assert.Equal(t, "test", tables[0], "table name")

	tables, err = db.Tables("temp")
	checkNoError(t, err, "error looking for tables: %s")
	assert.Equal(t, 0, len(tables), "table count")

	_, err = db.Tables("bim")
	assert.T(t, err != nil, "error expected")
	//println(err.Error())
}

func TestViews(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	views, err := db.Views("")
	checkNoError(t, err, "error looking for views: %s")
	assert.Equal(t, 0, len(views), "table count")
	err = db.FastExec("CREATE VIEW myview AS SELECT 1")
	checkNoError(t, err, "error creating view: %s")
	views, err = db.Views("main")
	checkNoError(t, err, "error looking for views: %s")
	assert.Equal(t, 1, len(views), "table count")
	assert.Equal(t, "myview", views[0], "table name")

	views, err = db.Views("temp")
	checkNoError(t, err, "error looking for views: %s")
	assert.Equal(t, 0, len(views), "table count")

	_, err = db.Views("bim")
	assert.T(t, err != nil)
}

func TestIndexes(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
	checkNoError(t, db.Exec("CREATE INDEX idx ON test(a_string)"), "%s")

	indexes, err := db.Indexes("")
	checkNoError(t, err, "error looking for indexes: %s")
	assert.Equal(t, 1, len(indexes), "index count")
	tbl, ok := indexes["idx"]
	assert.T(t, ok, "no index")
	assert.Equalf(t, "test", tbl, "got: %s; want: %s", tbl, "test")

	_, err = db.Indexes("main")
	checkNoError(t, err, "error looking for indexes: %s")

	_, err = db.Indexes("temp")
	checkNoError(t, err, "error looking for indexes: %s")

	_, err = db.Indexes("bim")
	assert.T(t, err != nil)
}

func TestColumns(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)

	columns, err := db.Columns("", "test")
	checkNoError(t, err, "error listing columns: %s")
	if len(columns) != 4 {
		t.Fatalf("got %d columns; want 4", len(columns))
	}
	column := columns[2]
	assert.Equal(t, "int_num", column.Name, "column name")

	_, err = db.Columns("main", "test")
	checkNoError(t, err, "error listing columns: %s")

	_, err = db.Columns("bim", "test")
	assert.T(t, err != nil, "expected error")
	//println(err.Error())

	_, err = db.Columns("", "bim")
	//assert.T(t, err != nil, "expected error")
	//println(err.Error())
}

func TestForeignKeys(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	err := db.Exec("CREATE TABLE parent (id INTEGER PRIMARY KEY NOT NULL);" +
		"CREATE TABLE child (id INTEGER PRIMARY KEY NOT NULL, parentId INTEGER, " +
		"FOREIGN KEY (parentId) REFERENCES parent(id));")
	checkNoError(t, err, "error creating tables: %s")
	fks, err := db.ForeignKeys("", "child")
	checkNoError(t, err, "error listing FKs: %s")
	if len(fks) != 1 {
		t.Fatalf("got %d FK(s); want 1", len(fks))
	}
	fk := fks[0]
	if fk.From[0] != "parentId" || fk.Table != "parent" || fk.To[0] != "id" {
		t.Errorf("unexpected FK data: %#v", fk)
	}

	_, err = db.ForeignKeys("main", "child")
	checkNoError(t, err, "error listing FKs: %s")

	_, err = db.ForeignKeys("bim", "child")
	assert.T(t, err != nil)
	//println(err.Error())

	_, err = db.ForeignKeys("", "bim")
	//assert.T(t, err != nil)
	//println(err.Error())
}

func TestTableIndexes(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
	createIndex(db, t)

	indexes, err := db.TableIndexes("", "test")
	checkNoError(t, err, "error listing indexes: %s")
	if len(indexes) != 1 {
		t.Fatalf("got %d index(es); want one", len(indexes))
	}
	index := indexes[0]
	assert.Equal(t, "test_index", index.Name, "index name")
	assert.T(t, !index.Unique, "expected index 'test_index' to be not unique")

	columns, err := db.IndexColumns("", "test_index")
	checkNoError(t, err, "error listing index columns: %s")
	if len(columns) != 1 {
		t.Fatalf("got %d column(s); want one", len(columns))
	}
	column := columns[0]
	assert.Equal(t, "a_string", column.Name, "column name")

	_, err = db.TableIndexes("main", "test")
	checkNoError(t, err, "error listing indexes: %s")
	_, err = db.IndexColumns("main", "test_index")
	checkNoError(t, err, "error listing index columns: %s")

	_, err = db.TableIndexes("bim", "test")
	assert.T(t, err != nil)
	//println(err.Error())

	_, err = db.IndexColumns("bim", "test_index")
	assert.T(t, err != nil)
	//println(err.Error())
}

func TestColumnTypeAffinity(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	checkNoError(t, db.FastExec("CREATE TABLE test (i INT, f REAL, n NUM, b BLOB, t TEXT, v);"), "%s")
	s, err := db.Prepare("SELECT i, f, n, b, t, v FROM test")
	checkNoError(t, err, "%s")
	defer checkFinalize(s, t)

	assert.Equal(t, Integral, s.ColumnTypeAffinity(0), "affinity")
	assert.Equal(t, Real, s.ColumnTypeAffinity(1), "affinity")
	assert.Equal(t, Numerical, s.ColumnTypeAffinity(2), "affinity")
	assert.Equal(t, None, s.ColumnTypeAffinity(3), "affinity")
	assert.Equal(t, Textual, s.ColumnTypeAffinity(4), "affinity")
	assert.Equal(t, None, s.ColumnTypeAffinity(5), "affinity")
}

func TestExpressionTypeAffinity(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	checkNoError(t, db.FastExec("CREATE TABLE test (i INT, f REAL, n NUM, b BLOB, t TEXT, v);"), "%s")
	s, err := db.Prepare("SELECT abs(i), f * 2, n / 2, lower(t) FROM test")
	checkNoError(t, err, "%s")
	defer checkFinalize(s, t)

	assert.Equal(t, None, s.ColumnTypeAffinity(0), "affinity")
	assert.Equal(t, None, s.ColumnTypeAffinity(1), "affinity")
	assert.Equal(t, None, s.ColumnTypeAffinity(2), "affinity")
	assert.Equal(t, None, s.ColumnTypeAffinity(3), "affinity")
}
