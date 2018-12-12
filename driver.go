// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"time"
)

func init() {
	sql.Register("sqlite3", &impl{open: defaultOpen})
	if os.Getenv("SQLITE_LOG") != "" {
		ConfigLog(func(d interface{}, err error, msg string) {
			log.Printf("%s: %s, %s\n", d, err, msg)
		}, "SQLITE")
	}
	ConfigMemStatus(false)
}

// impl is an adapter to database/sql/driver
type impl struct {
	open      func(name string) (*Conn, error)
	configure func(*Conn) error
}
type conn struct {
	c *Conn
}
type stmt struct {
	s            *Stmt
	rowsRef      bool // true if there is a rowsImpl associated to this statement that has not been closed.
	pendingClose bool
}
type rowsImpl struct {
	s           *stmt
	columnNames []string // cache
	ctx         context.Context
}

type result struct {
	id   int64
	rows int64
}

func (r *result) LastInsertId() (int64, error) {
	return r.id, nil
}

func (r *result) RowsAffected() (int64, error) {
	return r.rows, nil
}

// NewDriver creates a new driver with specialized connection creation/configuration.
//   NewDriver(customOpen, nil) // no post-creation hook
//   NewDriver(nil, customConfigure) // default connection creation but specific configuration step
func NewDriver(open func(name string) (*Conn, error), configure func(*Conn) error) driver.Driver {
	if open == nil {
		open = defaultOpen
	}
	return &impl{open: open, configure: configure}
}

var defaultOpen = func(name string) (*Conn, error) {
	// OpenNoMutex == multi-thread mode (http://sqlite.org/compile.html#threadsafe and http://sqlite.org/threadsafe.html)
	c, err := Open(name, OpenURI, OpenNoMutex, OpenReadWrite, OpenCreate)
	if err != nil {
		return nil, err
	}
	c.BusyTimeout(10 * time.Second)
	//c.DefaultTimeLayout = "2006-01-02 15:04:05.999999999"
	c.ScanNumericalAsTime = true
	return c, nil
}

// Open opens a new database connection.
// ":memory:" for memory db,
// "" for temp file db
func (d *impl) Open(name string) (driver.Conn, error) {
	c, err := d.open(name)
	if err != nil {
		return nil, err
	}
	if d.configure != nil {
		if err = d.configure(c); err != nil {
			_ = c.Close()
			return nil, err
		}
	}
	return &conn{c}, nil
}

// Unwrap gives access to underlying driver connection.
func Unwrap(db *sql.DB) *Conn {
	_, err := db.Exec("unwrap")
	if cerr, ok := err.(ConnError); ok {
		return cerr.c
	}
	return nil
}

func (c *conn) Ping(ctx context.Context) error {
	if c.c.IsClosed() {
		return driver.ErrBadConn
	}
	_, err := c.ExecContext(ctx, "PRAGMA schema_verion", []driver.NamedValue{})
	return err
}

// PRAGMA schema_version may be used to detect when the database schema is altered

func (c *conn) Exec(query string, args []driver.Value) (driver.Result, error) {
	panic("ExecContext was not called.")
}

func (c *conn) Prepare(query string) (driver.Stmt, error) {
	panic("use PrepareContext")
}

func (c *conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if c.c.IsClosed() {
		return nil, driver.ErrBadConn
	}
	s, err := c.c.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &stmt{s: s}, nil
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.c.IsClosed() {
		return nil, driver.ErrBadConn
	}
	if ctx.Done() != nil {
		c.c.ProgressHandler(progressHandler, 100, ctx)
		defer c.c.ProgressHandler(nil, 0, nil)
	}
	if len(args) == 0 {
		if query == "unwrap" {
			return nil, ConnError{c: c.c}
		}
		if err := c.c.FastExec(query); err != nil {
			return nil, ctxError(ctx, err)
		}
		return c.c.result(), nil
	}
	for len(query) > 0 {
		s, err := c.c.Prepare(query)
		if err != nil {
			return nil, ctxError(ctx, err)
		} else if s.stmt == nil {
			// this happens for a comment or white-space
			query = s.tail
			continue
		}
		var subargs []driver.NamedValue
		count := s.BindParameterCount()
		if len(s.tail) > 0 && len(args) >= count {
			subargs = args[:count]
			args = args[count:]
		} else {
			subargs = args
		}
		if err = s.bindNamedValue(subargs); err != nil {
			return nil, ctxError(ctx, err)
		}
		err = s.exec()
		if err != nil {
			s.finalize()
			return nil, ctxError(ctx, err)
		}
		if err = s.finalize(); err != nil {
			return nil, ctxError(ctx, err)
		}
		query = s.tail
	}
	return c.c.result(), nil
}

func (c *conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.c.IsClosed() {
		return nil, driver.ErrBadConn
	}
	st, err := c.c.Prepare(query)
	if err != nil {
		return nil, err
	}
	s := &stmt{s: st}
	return s.QueryContext(ctx, args)
}

func (c *conn) Close() error {
	return c.c.Close()
}

func (c *conn) Begin() (driver.Tx, error) {
	if c.c.IsClosed() {
		return nil, driver.ErrBadConn
	}
	if err := c.c.Begin(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if c.c.IsClosed() {
		return nil, driver.ErrBadConn
	}
	if !c.c.GetAutocommit() {
		return nil, errors.New("Nested transactions are not supported")
	}
	if err := c.c.SetQueryOnly("", opts.ReadOnly); err != nil {
		return nil, err
	}
	switch sql.IsolationLevel(opts.Isolation) {
	case sql.LevelDefault, sql.LevelSerializable:
		if err := c.c.FastExec("PRAGMA read_uncommitted=0"); err != nil {
			return nil, err
		}
	case sql.LevelReadUncommitted:
		if err := c.c.FastExec("PRAGMA read_uncommitted=1"); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("isolation level %d is not supported", opts.Isolation)
	}
	return c.Begin()
}

func (c *conn) Commit() error {
	return c.c.Commit()
}
func (c *conn) Rollback() error {
	return c.c.Rollback()
}

func (s *stmt) Close() error {
	if s.rowsRef { // Currently, it never happens because the sql.Stmt doesn't call driver.Stmt in this case
		s.pendingClose = true
		return nil
	}
	return s.s.Finalize()
}

func (s *stmt) NumInput() int {
	return s.s.BindParameterCount()
}

func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	panic("Using ExecContext")
}

func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	panic("Use QueryContext")
}

func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if err := s.s.bindNamedValue(args); err != nil {
		return nil, err
	}
	if ctx.Done() != nil {
		s.s.c.ProgressHandler(progressHandler, 100, ctx)
		defer s.s.c.ProgressHandler(nil, 0, nil)
	}
	if err := s.s.exec(); err != nil {
		return nil, ctxError(ctx, err)
	}
	return s.s.c.result(), nil
}

func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if s.rowsRef {
		return nil, errors.New("previously returned Rows still not closed")
	}
	if err := s.s.bindNamedValue(args); err != nil {
		return nil, err
	}
	s.rowsRef = true
	if ctx.Done() != nil {
		s.s.c.ProgressHandler(progressHandler, 100, ctx)
	}
	return &rowsImpl{s, nil, ctx}, nil
}

func (s *stmt) bind(args []driver.Value) error {
	for i, v := range args {
		if err := s.s.BindByIndex(i+1, v); err != nil {
			return err
		}
	}
	return nil
}

func (r *rowsImpl) Columns() []string {
	if r.columnNames == nil {
		r.columnNames = r.s.s.ColumnNames()
	}
	return r.columnNames
}

func (r *rowsImpl) Next(dest []driver.Value) error {
	ok, err := r.s.s.Next()
	if err != nil {
		return ctxError(r.ctx, err)
	}
	if !ok {
		return io.EOF
	}
	for i := range dest {
		dest[i], _ = r.s.s.ScanValue(i, true)
		/*if !driver.IsScanValue(dest[i]) {
			panic("Invalid type returned by ScanValue")
		}*/
	}
	return nil
}

func (r *rowsImpl) Close() error {
	if r.ctx.Done() != nil {
		r.s.s.c.ProgressHandler(nil, 0, nil)
	}
	r.s.rowsRef = false
	if r.s.pendingClose {
		return r.s.Close()
	}
	return r.s.s.Reset()
}

func (r *rowsImpl) HasNextResultSet() bool {
	return len(r.s.s.tail) > 0
}

func (r *rowsImpl) NextResultSet() error {
	currentStmt := r.s.s
	nextQuery := currentStmt.tail
	var nextStmt *Stmt
	var err error
	for len(nextQuery) > 0 {
		nextStmt, err = currentStmt.c.Prepare(nextQuery)
		if err != nil {
			return err
		} else if nextStmt.stmt == nil {
			// this happens for a comment or white-space
			nextQuery = nextStmt.tail
			continue
		}
		break
	}
	if nextStmt == nil {
		return io.EOF
	}
	// TODO close vs reset ?
	err = currentStmt.Finalize()
	if err != nil {
		return err
	}
	r.s.s = nextStmt
	return nil
}

func (r *rowsImpl) ColumnTypeScanType(index int) reflect.Type {
	switch r.s.s.ColumnType(index) {
	case Integer:
		return reflect.TypeOf(int64(0))
	case Float:
		return reflect.TypeOf(float64(0))
	case Text:
		return reflect.TypeOf("")
	case Null:
		return reflect.TypeOf(nil)
	case Blob:
		fallthrough
	default:
		return reflect.TypeOf([]byte{})
	}
}

func (r *rowsImpl) ColumnTypeDatabaseTypeName(index int) string {
	return r.s.s.ColumnDeclaredType(index)
}

func (c *Conn) result() driver.Result {
	// TODO How to know that the last Stmt has done an INSERT? An authorizer?
	id := c.LastInsertRowid()
	// TODO How to know that the last Stmt has done a DELETE/INSERT/UPDATE? An authorizer?
	rows := int64(c.Changes())
	return &result{id, rows} // FIXME RowAffected/noRows
}

func (s *Stmt) bindNamedValue(args []driver.NamedValue) error {
	for _, v := range args {
		if len(v.Name) == 0 {
			if err := s.BindByIndex(v.Ordinal, v.Value); err != nil {
				return err
			}
		} else {
			index, err := s.BindParameterIndex(":" + v.Name) // TODO "$" and "@"
			if err != nil {
				return err
			}
			if err = s.BindByIndex(index, v.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

func progressHandler(p interface{}) bool {
	if ctx, ok := p.(context.Context); ok {
		select {
		case <-ctx.Done():
			// Cancelled
			return true
		default:
			return false
		}
	}
	return false
}

func ctxError(ctx context.Context, err error) error {
	ctxErr := ctx.Err()
	if ctxErr != nil {
		return ctxErr
	}
	return err
}
