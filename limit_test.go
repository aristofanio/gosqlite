// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gosqlite

import (
	"testing"

	"github.com/bmizerany/assert"
	. "github.com/gwenn/gosqlite"
)

func TestLimit(t *testing.T) {
	db := open(t)
	defer checkClose(db, t)

	limitVariableNumber := db.Limit(LimitVariableNumber)
	assert.Tf(t, limitVariableNumber < 1e6, "unexpected value for LimitVariableNumber: %d", limitVariableNumber)
	oldLimitVariableNumber := db.SetLimit(LimitVariableNumber, 99)
	assert.Equalf(t, limitVariableNumber, oldLimitVariableNumber, "got LimitVariableNumber: %d; want %d", oldLimitVariableNumber, limitVariableNumber)
	limitVariableNumber = db.Limit(LimitVariableNumber)
	assert.Equalf(t, int32(99), limitVariableNumber, "got LimitVariableNumber: %d; want %d", limitVariableNumber, 99)

}
