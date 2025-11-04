// Copyright 2021, 2024 T-Mobile USA, Inc.
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
	"runtime/debug"
)

// PageError contains an error returned by the [PageGetter.GetPage]
// callback, along with the failing page request.
type PageError struct {
	PageRequest PageRequest // The request that failed
	Err         error       // The error that occurred
}

// Error returns the error message.
func (pe PageError) Error() string {
	return pe.Err.Error()
}

// Unwrap retrieves the underlying error.
func (pe PageError) Unwrap() error {
	return pe.Err
}

// PanicError is an error that represents a panic that occurred during
// go routine execution.
type PanicError struct {
	Panic any    // The panic value
	Trace string // The captured stack trace
	err   string // The constructed error message
}

// NewPanicError constructs a new PanicError instance with the
// specified panic value.  Constructs and saves a stack trace.
func NewPanicError(panicVal any) *PanicError {
	return &PanicError{
		Panic: panicVal,
		Trace: string(debug.Stack()),
	}
}

// Error returns the error message.
func (pe *PanicError) Error() string {
	// Have we already constructed the error message?
	if pe.err == "" {
		// Construct the error message
		switch p := pe.Panic.(type) {
		case error:
			pe.err = fmt.Sprintf("panic encountered: error %q with stack %s", p.Error(), pe.Trace)
		case string:
			pe.err = fmt.Sprintf("panic encountered: string value %q with stack %s", p, pe.Trace)
		case fmt.Stringer:
			pe.err = fmt.Sprintf("panic encountered: value %q with stack %s", p.String(), pe.Trace)
		default:
			pe.err = fmt.Sprintf("panic encountered: value %#v with stack %s", pe.Panic, pe.Trace)
		}
	}

	return pe.err
}

// catchPanic0 is a helper that calls a function returning no
// arguments and catches any panic, returning a [PanicErorr] if a
// panic occurs.
func catchPanic0(fn func()) (perr error) {
	defer func() {
		if r := recover(); r != nil {
			perr = NewPanicError(r)
		}
	}()
	fn()
	return nil
}

// catchPanic2 is a helper that calls a function returning a value of
// some type and an error and catches any panic, ensuring a
// [PanicError] is returned if a panic occurs.
func catchPanic2[T any](fn func() (T, error)) (result T, perr error) {
	defer func() {
		if r := recover(); r != nil {
			perr = NewPanicError(r)
		}
	}()
	return fn()
}
