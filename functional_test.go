// Copyright 2024 T-Mobile USA, Inc.
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
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Number of times to run tests; running the tests multiple times
// should help catch any race conditions or ordering errors
const TestCount = 5

type PagedData struct {
	data        []string // Actual data
	perPage     int      // Number of results per page
	reportItems bool     // Report TotalItems
	reportPages bool     // Report TotalPages
	pageAhead   int      // Number of page requests to create
}

func (pd PagedData) GetPage(_ context.Context, depag State, req PageRequest) ([]string, error) {
	// First, update the items and pages
	totalItems := 0
	if pd.reportItems {
		totalItems = len(pd.data)
	}
	totalPages := 0
	if pd.reportPages {
		totalPages = len(pd.data) / pd.perPage
		if len(pd.data)%pd.perPage == 0 {
			totalPages++
		}
	}
	depag.Update(TotalItems(totalItems), TotalPages(totalPages), PerPage(pd.perPage))

	// Next, generate the page requests
	maxPage := pd.pageAhead
	if maxPage < 1 {
		maxPage = 1
	}
	for i := req.PageIndex + 1; i <= maxPage; i++ {
		depag.Request(i, nil)
	}

	// Now generate and return a page
	if req.PageIndex*pd.perPage >= len(pd.data) {
		return nil, nil
	}
	subset := pd.data[req.PageIndex*pd.perPage:]
	pageLen := pd.perPage
	if len(subset) < pageLen {
		pageLen = len(subset)
	}
	dest := make([]string, pageLen)
	copy(dest, subset)
	return dest, nil
}

func TestBasicFunction(t *testing.T) {
	// Run the test several times to try to tickle any race conditions
	// or similar errors
	for i := 0; i < TestCount; i++ {
		t.Run(fmt.Sprintf("basic-%d", i), func(t *testing.T) {
			ctx := context.Background()
			data := PagedData{
				data: []string{
					"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
				},
				perPage:   3,
				pageAhead: 5,
			}
			result := &ListHandler[string]{}

			d := Depaginate[string](ctx, data, result)
			err := d.Wait()

			assert.NoError(t, err)
			assert.Equal(t, data.data, result.Items)
		})
	}
}

func TestAppendFunction(t *testing.T) {
	// Run the test several times to try to tickle any race conditions
	// or similar errors
	for i := 0; i < TestCount; i++ {
		t.Run(fmt.Sprintf("append-%d", i), func(t *testing.T) {
			ctx := context.Background()
			data := PagedData{
				data: []string{
					"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
				},
				perPage:   3,
				pageAhead: 5,
			}
			result := &ListHandler[string]{}

			d := Depaginate[string](ctx, data, result)
			err := d.Wait()

			assert.NoError(t, err)

			d = Depaginate[string](ctx, data, result)
			err = d.Wait()

			assert.NoError(t, err)
			assert.Equal(t, append(data.data, data.data...), result.Items)
		})
	}
}
