// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"fmt"
	"testing"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func init() {
	err := ConfigThreadingMode(Serialized)
	if err != nil {
		panic(fmt.Sprintf("cannot change threading mode: '%s'", err))
	}
	err = ConfigMemStatus(true)
	if err != nil {
		panic(fmt.Sprintf("cannot activate mem status: '%s'", err))
	}
	err = ConfigURI(true)
	if err != nil {
		panic(fmt.Sprintf("cannot activate uri handling: '%s'", err))
	}
}

func TestEnableFKey(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	b, err := db.IsFKeyEnabled()
	checkNoError(t, err, "%s")
	if !b {
		b, err = db.EnableFKey(true)
		checkNoError(t, err, "%s")
		assert.T(t, b, "cannot enable FK")
	}
}

func TestEnableTriggers(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	b, err := db.AreTriggersEnabled()
	checkNoError(t, err, "%s")
	//if !b {
	b, err = db.EnableTriggers(true)
	checkNoError(t, err, "%s")
	assert.T(t, b, "cannot enable triggers")
	//}
}

func TestEnableExtendedResultCodes(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	checkNoError(t, db.EnableExtendedResultCodes(true), "cannot enable extended result codes: %s")
}

func TestConnSettings(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)
	err := db.SetRecursiveTriggers("main", true)
	checkNoError(t, err, "SetRecursiveTriggers error: %s")
}

func TestCompileOptionUsed(t *testing.T) {
	b := CompileOptionUsed("SQLITE_ENABLE_COLUMN_METADATA")
	if !b {
		t.Log("COLUMN_METADATA disabled")
	}
	//assert.T(t, b, "COLUMN_METADATA disabled")
}
