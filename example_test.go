// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite_test

import (
	"fmt"
	"github.com/gwenn/gosqlite"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func Example() {
	db, err := sqlite.Open(":memory:") // path to db or "" for temp db
	check(err)
	defer db.Close()
	err = db.Exec("CREATE TABLE test(id INTEGER PRIMARY KEY NOT NULL, name TEXT NOT NULL UNIQUE); -- ... and other ddls separated by semi-colon")
	check(err)
	ins, err := db.Prepare("INSERT INTO test (name) VALUES (?)")
	check(err)
	defer ins.Finalize()
	_, err = ins.Insert("gosqlite driver")
	check(err)
	s, err := db.Prepare("SELECT name from test WHERE name like ?", "%go%")
	check(err)
	defer s.Finalize()
	var name string
	err = s.Select(func(s *sqlite.Stmt) (err error) {
		if err = s.Scan(&name); err != nil {
			return
		}
		fmt.Printf("%s\n", name)
		return
	})
	// Output: gosqlite driver
}

func ExampleOpen() {
	db, err := sqlite.Open(":memory:")
	check(err)
	defer db.Close()
	fmt.Printf("db path: %q\n", db.Filename("main"))
	// Output: db path: ""
}

func ExampleConn_Exec() {
	db, err := sqlite.Open(":memory:")
	check(err)
	defer db.Close()

	err = db.Exec("CREATE TABLE test1 (content TEXT); CREATE TABLE test2 (content TEXT); INSERT INTO test1 VALUES ('DATA')")
	check(err)
	tables, err := db.Tables("")
	check(err)
	fmt.Printf("%d tables\n", len(tables))
	// Output: 2 tables
}

func ExampleStmt_ExecDml() {
	db, err := sqlite.Open(":memory:")
	check(err)
	defer db.Close()
	err = db.Exec("CREATE TABLE test (content TEXT); INSERT INTO test VALUES ('Go'); INSERT INTO test VALUES ('SQLite')")
	check(err)

	s, err := db.Prepare("UPDATE test SET content = content || 'lang' where content like ?")
	check(err)
	defer s.Finalize()
	changes, err := s.ExecDml("%o")
	check(err)
	fmt.Printf("%d change(s)\n", changes)
	// Output: 1 change(s)
}

func ExampleStmt_Insert() {
	db, err := sqlite.Open(":memory:")
	check(err)
	defer db.Close()
	err = db.Exec("CREATE TABLE test (content TEXT)")
	check(err)

	s, err := db.Prepare("INSERT INTO test VALUES (?)")
	check(err)
	defer s.Finalize()
	data := []string{"Go", "SQLite", "Driver"}
	for _, d := range data {
		rowId, err := s.Insert(d)
		check(err)
		fmt.Printf("%d\n", rowId)
	}
	// Output: 1
	// 2
	// 3
}

func ExampleStmt_NamedScan() {
	db, err := sqlite.Open(":memory:")
	check(err)
	defer db.Close()

	s, err := db.Prepare("SELECT 1 as id, 'Go' as name UNION SELECT 2, 'SQLite'")
	check(err)
	defer s.Finalize()

	var id int
	var name string
	err = s.Select(func(s *sqlite.Stmt) (err error) {
		if err = s.NamedScan("name", &name, "id", &id); err != nil {
			return
		}
		fmt.Println(id, name)
		return
	})
	// Output: 1 Go
	// 2 SQLite
}

func ExampleStmt_Scan() {
	db, err := sqlite.Open(":memory:")
	check(err)
	defer db.Close()

	s, err := db.Prepare("SELECT 1 as id, 'Go' as name, 'Y' as status UNION SELECT 2, 'SQLite', 'yes'")
	check(err)
	defer s.Finalize()

	var id int
	var name string
	var status bool

	converter := func(value interface{}) (bool, error) {
		status = value == "Y" || value == "yes"
		return false, nil
	}

	err = s.Select(func(s *sqlite.Stmt) (err error) {
		if err = s.Scan(&id, &name, converter); err != nil {
			return
		}
		fmt.Println(id, name, status)
		return
	})
	// Output: 1 Go true
	// 2 SQLite true
}

func ExampleNewBackup() {
	dst, err := sqlite.Open(":memory:")
	check(err)
	defer dst.Close()
	src, err := sqlite.Open(":memory:")
	check(err)
	defer src.Close()
	err = src.Exec("CREATE TABLE test (content BLOB); INSERT INTO test VALUES (zeroblob(100))")
	check(err)

	bck, err := sqlite.NewBackup(dst, "main", src, "main")
	check(err)
	defer bck.Close()

	cbs := make(chan sqlite.BackupStatus)
	ack := make(chan bool)
	go func() {
		for {
			s := <-cbs
			fmt.Printf("backup progress (remaining: %d)\n", s.Remaining)
			ack <- true
		}
	}()
	err = bck.Run(100, 250000, cbs)
	check(err)
	<-ack
	// Output: backup progress (remaining: 0)
}

func ExampleConn_NewBlobReader() {
	db, err := sqlite.Open(":memory:")
	check(err)
	err = db.Exec("CREATE TABLE test (content BLOB); INSERT INTO test VALUES (zeroblob(10))")
	check(err)
	rowid := db.LastInsertRowid()

	br, err := db.NewBlobReader("main", "test", "content", rowid)
	check(err)
	defer br.Close()
	size, err := br.Size()
	check(err)
	// TODO A real 'incremental' example...
	content := make([]byte, size)
	n, err := br.Read(content)
	check(err)
	fmt.Printf("blob (size: %d, read: %d, content: %q\n", size, n, content)
	// Output: blob (size: 10, read: 10, content: "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
}
