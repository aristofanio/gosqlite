// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build all

package gosqlite

import (
	"testing"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func TestColumn(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)

	column, err := db.Column("", "test", "id")
	checkNoError(t, err, "error getting column metadata: %s")
	assert.Equal(t, "id", column.Name, "column name")
	assert.Equal(t, 1, column.Pk, "primary key index")
	assert.T(t, !column.Autoinc, "expecting autoinc flag to be false")

	column, err = db.Column("main", "test", "id")
	checkNoError(t, err, "error getting column metadata: %s")

	column, err = db.Column("", "test", "bim")
	assert.T(t, err != nil, "expected error")
	assert.T(t, err.Error() != "")
	//println(err.Error())
}

func TestColumnMetadata(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	s, err := db.Prepare("SELECT name AS table_name FROM sqlite_master")
	check(err)
	defer checkFinalize(s, t)

	databaseName := s.ColumnDatabaseName(0)
	assert.Equal(t, "main", databaseName, "database name")
	tableName := s.ColumnTableName(0)
	assert.Equal(t, "sqlite_master", tableName, "table name")
	originName := s.ColumnOriginName(0)
	assert.Equal(t, "name", originName, "origin name")
	declType := s.ColumnDeclaredType(0)
	assert.Equal(t, "text", declType, "declared type")
	affinity := s.ColumnTypeAffinity(0)
	assert.Equal(t, Textual, affinity, "affinity")
}

func TestColumnMetadataOnView(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	createTable(db, t)
	err := db.FastExec("CREATE VIEW vtest AS SELECT * FROM test")
	checkNoError(t, err, "error creating view: %s")

	s, err := db.Prepare("SELECT a_string AS str FROM vtest")
	check(err)
	defer checkFinalize(s, t)

	databaseName := s.ColumnDatabaseName(0)
	assert.Equal(t, "main", databaseName, "database name")
	tableName := s.ColumnTableName(0)
	assert.Equal(t, "test", tableName, "table name")
	originName := s.ColumnOriginName(0)
	assert.Equal(t, "a_string", originName, "origin name")
	declType := s.ColumnDeclaredType(0)
	assert.Equal(t, "TEXT", declType, "declared type")
	affinity := s.ColumnTypeAffinity(0)
	assert.Equal(t, Textual, affinity, "affinity")
}

func TestColumnMetadataOnExpr(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.FastExec("CREATE VIEW vtest AS SELECT date('now') as tic")
	checkNoError(t, err, "error creating view: %s")

	s, err := db.Prepare("SELECT tic FROM vtest")
	check(err)
	defer checkFinalize(s, t)

	databaseName := s.ColumnDatabaseName(0)
	assert.Equal(t, "", databaseName, "database name")
	tableName := s.ColumnTableName(0)
	assert.Equal(t, "", tableName, "table name")
	originName := s.ColumnOriginName(0)
	assert.Equal(t, "", originName, "origin name")
	declType := s.ColumnDeclaredType(0)
	assert.Equal(t, "", declType, "declared type")
	affinity := s.ColumnTypeAffinity(0)
	assert.Equal(t, None, affinity, "affinity")
}
