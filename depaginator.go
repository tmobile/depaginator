// Copyright (c) 2021 T-Mobile
//
// Licensed under the Apache License, Version 2.0 (the "License"); you
// may not use this file except in compliance with the License.  You
// may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.  See the License for the specific language governing
// permissions and limitations under the License.

// Package depaginator contains an implementation of a function,
// Depaginate, that allows iterating over all items in a paginated
// API.  The consuming application needs to implement the API and Page
// interfaces and pass Depaginate a context.Context, the API, and an
// initial page request, then call Wait on the result; the Depaginator
// will then take care of the rest, calling the API.GetPage method to
// retrieve pages of results; the API.HandleItem method to handle the
// found items; and API.Done when the iteration is complete.
package depaginator

import (
	"context"
	"errors"
	"sync"
)

// API is the type that is passed to Depaginate.  Methods on this type
// are called to get pages and handle items from the pages.
type API interface {
	// GetPage is a page retriever function.  It takes a PageMeta
	// object, which it should fill in, and a PageRequest object, and
	// returns a Page object and an error.
	GetPage(ctx context.Context, pm *PageMeta, req PageRequest) (Page, error)

	// HandleItem is a function called with an item by the
	// Depaginator.
	HandleItem(ctx context.Context, idx int, item interface{})

	// Done is a function called when the Depaginator is done.  It is
	// called by Depaginator.Wait.
	Done(pm PageMeta)
}

// Depaginate is a tool for iterating over all items in a paginated
// response.  It uses goroutines to perform its work, and is capable
// of issuing requests for every available page simultaneously, so
// callers should ensure GetPage routine passed to Depaginator
// incorporates some sort of limiter to ensure they don't overwhelm
// any rate limits that may be set on the API.
func Depaginate(ctx context.Context, api API, req PageRequest) *Depaginator {
	dp := &Depaginator{
		meta:      &PageMeta{},
		api:       api,
		cancelers: map[int]context.CancelFunc{},
		pages:     &pageMap{},
		wg:        &sync.WaitGroup{},
	}

	// Get the first page
	dp.issueRequests(ctx, []PageRequest{req})

	return dp
}

// Depaginator is returned by the Depaginate function to allow the
// caller to wait for the iteration to complete.
type Depaginator struct {
	sync.Mutex                            // Protects the set of errors and page metadata
	errors     []PageError                // The errors encountered
	meta       *PageMeta                  // Page metadata
	api        API                        // The API to use for depagination
	cancelers  map[int]context.CancelFunc // Mapping of page index to cancel func
	pages      *pageMap                   // Bitmap of requested pages
	wg         *sync.WaitGroup            // A wait group for Wait to wait upon
}

// Wait waits for the iteration to complete.  It returns an unordered
// list of the errors encountered during the iteration.  Each entry is
// a PageError; this implements the error interface, and bundles
// together the error that occurred along with the page request that
// resulted in the error.
func (dp *Depaginator) Wait() []PageError {
	dp.wg.Wait()
	dp.api.Done(*dp.meta)

	return dp.errors
}

// registerCanceler registers a canceler for the page context and
// retrieves the current page metadata.
func (dp *Depaginator) registerCanceler(idx int, cancelFn context.CancelFunc) PageMeta {
	dp.Lock()
	defer dp.Unlock()

	// Register the canceler
	dp.cancelers[idx] = cancelFn

	// Prepare the page meta
	meta := *dp.meta

	return meta
}

// cancelPages cancels pages with an index greater than the one
// specified.  This method must be called with the mutex locked.
func (dp *Depaginator) cancelPages(idx int) {
	for page, canceler := range dp.cancelers {
		if page > idx {
			canceler()
		}
	}
}

// issueRequests issues the specified page requests.  This method must
// be called with the mutex locked.
func (dp *Depaginator) issueRequests(ctx context.Context, reqs []PageRequest) {
	for _, req := range reqs {
		// Skip requests for pages we know don't exist
		if dp.meta.PageCount > 0 && req.PageIndex >= dp.meta.PageCount {
			continue
		}
		if !dp.pages.CheckAndSet(req.PageIndex) {
			dp.wg.Add(1)
			go dp.getPage(ctx, req)
		}
	}
}

// pageError is called if the page getter returned an error.  This
// method must be called with the mutex locked.
func (dp *Depaginator) pageError(req PageRequest, err error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return
	}

	dp.errors = append(dp.errors, PageError{
		PageRequest: req,
		Err:         err,
	})
}

// handle handles a single item.  This method must be called with the
// mutex unlocked.
func (dp *Depaginator) handle(ctx context.Context, idx int, item interface{}) {
	defer dp.wg.Done()
	dp.api.HandleItem(ctx, idx, item)
}

// getPage requests a page and iterates over retrieved pages and
// items.
func (dp *Depaginator) getPage(ctx context.Context, req PageRequest) {
	defer dp.wg.Done()

	// First, construct the child context
	childCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	// Construct the page meta and save the canceler
	meta := dp.registerCanceler(req.PageIndex, cancelFn)

	// Get the page
	page, err := dp.api.GetPage(childCtx, &meta, req)

	// Lock the object to safely handle the result
	dp.Lock()
	defer dp.Unlock()

	// Withdraw the canceler
	delete(dp.cancelers, req.PageIndex)

	// Did we get an error?
	if err != nil {
		dp.pageError(req, err)
		return
	}

	// Check for any metadata updates
	dp.meta.update(meta)

	// Are there returned requests?
	if meta.Requests == nil || len(meta.Requests) == 0 || page.Len() < dp.meta.PerPage {
		// No more pages
		dp.meta.SetPageCount(req.PageIndex + 1)
		dp.meta.SetItemCount(dp.meta.PerPage*req.PageIndex + page.Len())
		dp.cancelPages(req.PageIndex)
	} else {
		// Issue requests for the new pages
		dp.issueRequests(ctx, meta.Requests)
	}

	// Now handle the items
	itemBase := dp.meta.PerPage * req.PageIndex
	for i := 0; i < page.Len(); i++ {
		dp.wg.Add(1)
		go dp.handle(ctx, itemBase+i, page.Get(i))
	}
}
