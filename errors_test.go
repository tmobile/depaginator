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
	"testing"

	"github.com/stretchr/testify/assert"
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
