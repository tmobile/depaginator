// Copyright 2021 T-Mobile USA, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// See the LICENSE file for additional language around the disclaimer of warranties.
// Trademark Disclaimer: Neither the name of “T-Mobile, USA” nor the names of
// its contributors may be used to endorse or promote products

package depaginator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageErrorError(t *testing.T) {
	obj := PageError{
		Err: assert.AnError,
	}

	result := obj.Error()

	assert.Equal(t, assert.AnError.Error(), result)
}

func TestPageErrorUnwrap(t *testing.T) {
	obj := PageError{
		Err: assert.AnError,
	}

	result := obj.Unwrap()

	assert.Same(t, assert.AnError, result)
}

func TestPanicErrorImplementsError(t *testing.T) {
	assert.Implements(t, (*error)(nil), &PanicError{})
}

func TestNewPanicError(t *testing.T) {
	panicVal := "test panic"

	result := NewPanicError(panicVal)

	assert.Equal(t, panicVal, result.Panic)
	assert.NotEmpty(t, result.Trace)
}

func TestPanicErrorErrorBase(t *testing.T) {
	obj := &PanicError{
		Panic: assert.AnError,
		Trace: "test trace",
		err:   "cached",
	}

	result := obj.Error()

	assert.Equal(t, "cached", result)
	assert.Equal(t, "cached", obj.err)
}

func TestPanicErrorErrorError(t *testing.T) {
	obj := &PanicError{
		Panic: assert.AnError,
		Trace: "test trace",
	}

	result := obj.Error()

	assert.Equal(t, fmt.Sprintf("panic encountered: error %q with stack test trace", assert.AnError.Error()), result)
	assert.Equal(t, result, obj.err)
}

func TestPanicErrorErrorString(t *testing.T) {
	obj := &PanicError{
		Panic: "some panic",
		Trace: "test trace",
	}

	result := obj.Error()

	assert.Equal(t, "panic encountered: string value \"some panic\" with stack test trace", result)
	assert.Equal(t, result, obj.err)
}

type typeForStringer struct{}

func (t typeForStringer) String() string {
	return "stringer value"
}

func TestPanicErrorErrorStringer(t *testing.T) {
	obj := &PanicError{
		Panic: typeForStringer{},
		Trace: "test trace",
	}

	result := obj.Error()

	assert.Equal(t, "panic encountered: value \"stringer value\" with stack test trace", result)
	assert.Equal(t, result, obj.err)
}

func TestPanicErrorErrorOther(t *testing.T) {
	obj := &PanicError{
		Panic: 17,
		Trace: "test trace",
	}

	result := obj.Error()

	assert.Equal(t, "panic encountered: value 17 with stack test trace", result)
	assert.Equal(t, result, obj.err)
}

func TestCatchPanic0Base(t *testing.T) {
	called := false
	fn := func() {
		called = true
	}

	err := catchPanic0(fn)

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestCatchPanic0WithPanic(t *testing.T) {
	called := false
	fn := func() {
		called = true
		panic("test panic")
	}

	err := catchPanic0(fn)

	var perr *PanicError
	require.ErrorAs(t, err, &perr)
	assert.Equal(t, "test panic", perr.Panic)
	assert.True(t, called)
}

func TestCatchPanic2Base(t *testing.T) {
	called := false
	fn := func() (string, error) {
		called = true
		return "result", assert.AnError
	}

	result, err := catchPanic2(fn)

	assert.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, "result", result)
	assert.True(t, called)
}

func TestCatchPanic2WithPanic(t *testing.T) {
	called := false
	fn := func() (string, error) {
		called = true
		panic("test panic")
	}

	result, err := catchPanic2(fn)

	var perr *PanicError
	require.ErrorAs(t, err, &perr)
	assert.Equal(t, "test panic", perr.Panic)
	assert.Equal(t, "", result)
	assert.True(t, called)
}
