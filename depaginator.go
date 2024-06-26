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

// Package depaginator contains an implementation of a function,
// [Depaginate], that allows iterating over all items in a paginated
// API.  The consuming application needs to pass [Depaginate] a
// [context.Context], a [PageGetter] page retriever, and an item
// [Handler], then call [Depaginator.Wait] on the result; the
// [Depaginator] will then take care of the rest, calling the
// [PageGetter.GetPage] method to retrieve pages of results and the
// [Handler.Handle] method to handle the found items.  Options exist
// to call a [Starter.Start] method before beginning, [Updater.Update]
// method when the total number of pages or items is discovered; and
// [Doner.Done] when the iteration is complete.
package depaginator

import (
	"context"
	"errors"
	"sync"
)

// PageRequest describes a request for a specific page.  Most of the
// page request lives in the Request field, which can be anything
// needed by the application; the only other field is the PageIndex
// field, which identifies the index of the page.  Note that PageIndex
// is 0-based.
type PageRequest struct {
	PageIndex int // The index of the page
	Request   any // The actual data needed to request the page
}

// Depaginator is returned by the [Depaginate] function to allow the
// caller to wait for the iteration to complete.  This object is also
// passed to [PageGetter.GetPage], and may be used to call
// [Depaginator.Update] and [Depaginator.Request] to update the number
// of items/pages or to request fetching additional pages,
// respectively.
type Depaginator[T any] struct {
	ctx        context.Context // A context for calls
	errors     []error         // Errors encountered
	totalItems int             // Total number of items
	totalPages int             // Total number of pages
	perPage    int             // Items per page
	pager      PageGetter[T]   // Object to retrieve pages with
	handler    Handler[T]      // Object to use to handle items
	starter    Starter         // Optional object to start iteration
	updater    Updater         // Optional object to notify updates to items/pages
	doner      Doner           // Optional object to notify end iteration

	cancelers map[int]context.CancelFunc // Mapping of page index to cancel function
	pages     *pageMap                   // Bitmap of requested pages
	wg        *sync.WaitGroup            // A wait group for Wait to wait upon
	updates   chan update[T]             // Updates to process
	done      chan struct{}              // Used to signal the daemon has exited
}

// Depaginate is a tool for iterating over all items in a paginated
// response.  It uses goroutines to perform its work, and is capable
// of issuing requests for every available page simultaneously, so
// callers should ensure the [PageGetter.GetPage] routine passed to
// Depaginate incorporates some sort of limiter to ensure they don't
// overwhelm any rate limits that may be set on the target API.  The
// [Handler.Handle] method will be called for each item.  Note that
// Depaginate returns a [Depaginator], and the calling application is
// expected to call [Depaginator.Wait].
func Depaginate[T any](ctx context.Context, pager PageGetter[T], handler Handler[T], opts ...Option) *Depaginator[T] {
	// Prepare the options
	o := options{
		capacity: DefaultCapacity,
	}
	if tmp, ok := handler.(Starter); ok {
		o.starter = tmp
	}
	if tmp, ok := handler.(Updater); ok {
		o.updater = tmp
	}
	if tmp, ok := handler.(Doner); ok {
		o.doner = tmp
	}

	// Parse the provided options
	for _, opt := range opts {
		opt.apply(&o)
	}

	// Construct the depaginator
	dp := &Depaginator[T]{
		ctx:        ctx,
		pager:      pager,
		totalItems: o.totalItems,
		totalPages: o.totalPages,
		perPage:    o.perPage,
		handler:    handler,
		starter:    o.starter,
		updater:    o.updater,
		doner:      o.doner,
		cancelers:  map[int]context.CancelFunc{},
		pages:      &pageMap{},
		wg:         &sync.WaitGroup{},
		updates:    make(chan update[T], o.capacity),
		done:       make(chan struct{}),
	}

	// Initialize the handler if required
	if dp.starter != nil {
		dp.starter.Start(ctx, dp.totalItems, dp.totalPages, dp.perPage)
	}

	// Issue the first request; can't use Depaginator.Request because
	// of a race: the update could be sitting in the queue, not yet
	// processed by the daemon, and Depaginator.Wait could be called.
	pageRequest[T]{
		idx: 0,
		req: o.initReq,
	}.applyUpdate(dp)

	// Start the daemon
	go dp.daemon()

	return dp
}

// daemon is the goroutine that processes updates from the
// [PageGetter.GetPage] methods.
func (dp *Depaginator[T]) daemon() {
	defer close(dp.done)
	for u := range dp.updates {
		// Save original metadata
		origItems, origPages, origPer := dp.totalItems, dp.totalPages, dp.perPage

		// Apply the update
		u.applyUpdate(dp)

		// If there were any changes, call the updater
		if dp.updater != nil && (origItems != dp.totalItems || origPages != dp.totalPages || origPer != dp.perPage) {
			dp.updater.Update(dp.ctx, dp.totalItems, dp.totalPages, dp.perPage)
		}
	}
}

// Wait waits for the iteration to complete.  It returns the errors
// encountered during the iteration, wrapped by [errors.Join].  Each
// error in the list is a [PageError], which bundles together the
// error and the corresponding page request.
func (dp *Depaginator[T]) Wait() error {
	// Wait for the pages and items
	dp.wg.Wait()

	// Signal the daemon to finish up
	close(dp.updates)
	<-dp.done

	// Call the doner
	if dp.doner != nil {
		dp.doner.Done(dp.ctx, dp.totalItems, dp.totalPages, dp.perPage)
	}

	return errors.Join(dp.errors...)
}

// update sends an update to the daemon.
func (dp *Depaginator[T]) update(update update[T]) {
	dp.updates <- update
}

// getPage is a wrapper around [PageGetter.GetPage] that implements
// the processing required to perform the depagination.
func (dp *Depaginator[T]) getPage(req PageRequest) {
	// Note: getPage is not complete until all its updates are
	// complete, so we use an update object to update the wait group
	defer dp.update(pageDone[T]{})

	// First, construct the child context
	childCtx, cancelFn := context.WithCancel(dp.ctx)
	defer cancelFn()

	// Register the canceler
	dp.update(cancelerFor[T]{
		page:     req.PageIndex,
		cancelFn: cancelFn,
	})

	// Get the page
	page, err := dp.pager.GetPage(childCtx, dp, req)

	// Withdraw the canceler
	dp.update(withdrawCanceler[T](req.PageIndex))

	// If there was an error, save it
	if err != nil {
		dp.update(errorSaver[T]{
			req: req,
			err: err,
		})
		return
	}

	// Handle the items
	dp.update(itemHandler[T]{
		idx:  req.PageIndex,
		page: page,
	})
}

// Update allows updating the total number of items, total number of
// pages, or the items per page.  The arguments passed to Update
// should be [TotalItems], [TotalPages], or [PerPage]; any other
// argument types will be ignored.
func (dp *Depaginator[T]) Update(updates ...any) {
	ups := bundle[T]{}
	for _, u := range updates {
		switch update := u.(type) {
		case TotalItems:
			ups = append(ups, totalItems[T](int(update)))
		case TotalPages:
			ups = append(ups, totalPages[T](int(update)))
		case PerPage:
			ups = append(ups, perPage[T](int(update)))
		}
	}

	if len(ups) > 0 {
		dp.update(ups)
	}
}

// Request requests the [Depaginator] retrieve a page.  Note that the
// page index is 0-based; the first page always has index 0.  The
// request is optional, and can contain any page-specific data, such
// as a page link.  Duplicate page requests are ignored, as is any
// request with an index greater than the total number of pages (if
// known).
func (dp *Depaginator[T]) Request(idx int, req any) {
	dp.update(pageRequest[T]{
		idx: idx,
		req: req,
	})
}

// PerPage retrieves the configured "per page" value for
// [Depaginator].  This allows a consumer to set the number of items
// per page when calling [Depaginate] (using the [PerPage] option).
// Applications should be careful to not mix this functionality with
// dynamic collection of the "per page" value, as the value is not
// protected by any mutex; if using this method, avoid passing
// [PerPage] to [Depaginator.Update] and arrange for a reasonable
// default if [PerPage] is not passed to [Depaginate] (in which case,
// this method will return 0).
func (dp *Depaginator[T]) PerPage() int {
	return dp.perPage
}
