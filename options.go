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
	"errors"
)

// DefaultCapacity is the default capacity for the updates channel.
const DefaultCapacity = 500

// options describes options for [Depaginate].
type options struct {
	totalItems int     // Total number of items (hint)
	totalPages int     // Total number of pages (hint)
	perPage    int     // Number of items per page
	capacity   int     // Capacity of the update queue
	starter    Starter // Object with a Start method
	updater    Updater // Object with an Update method
	doner      Doner   // Object with a Done method
	initReq    any     // Initial request
}

// Option describes an option that may be passed to [Depaginate].
type Option interface {
	// apply applies an option.
	apply(opts *options)
}

// TotalItems is used to indicate an update to the total number of
// items to be expected.  It may also be passed to [Depaginate] to
// hint to the total number of items to be expected.
type TotalItems int

// apply applies an option.
func (o TotalItems) apply(opts *options) {
	opts.totalItems = int(o)
}

// TotalPages is used to indicate an update to the total number of
// pages to be expected.  It may also be passed to [Depaginate] to
// hint to the total number of pages to be expected.
type TotalPages int

// apply applies an option.
func (o TotalPages) apply(opts *options) {
	opts.totalPages = int(o)
}

// PerPage is used to indicate an update to the number of items per
// page to be expected.  It may also be passed to [Depaginate] to hint
// to the number of items per page to be expected.
type PerPage int

// apply applies an option.
func (o PerPage) apply(opts *options) {
	opts.perPage = int(o)
}

// Capacity may be passed to [Depaginate] to control the size of the
// updates queue on the [Depaginator].  This defaults to
// [DefaultCapacity], which is set to a generous size.  Applications
// should only need to use this option if the default is insufficient
// for efficient operation.
type Capacity int

// apply applies an option.
func (o Capacity) apply(opts *options) {
	opts.capacity = int(o)
}

// WithStarterOption is an [Option] implementation that explicitly
// sets the [Starter] to use.
type WithStarterOption struct {
	starter Starter
}

// apply applies an option.
func (o WithStarterOption) apply(opts *options) {
	opts.starter = o.starter
}

// WithStarter returns an [Option] that can be passed to [Depaginate]
// which sets an [Starter] to be called when [Depaginate] begins its
// work.  The [Starter.Start] method is called with the initial values
// for total pages, total items, and per-page.  The default is the
// [Handler], if it implements [Starter].
func WithStarter(starter Starter) WithStarterOption {
	return WithStarterOption{
		starter: starter,
	}
}

// WithUpdaterOption is an [Option] implementation that explicitly
// sets the [Updater] to use.
type WithUpdaterOption struct {
	updater Updater
}

// apply applies an option.
func (o WithUpdaterOption) apply(opts *options) {
	opts.updater = o.updater
}

// WithUpdater returns an [Option] that can be passed to [Depaginate]
// which sets an [Updater] to be called when the total pages, total
// items, or per-page values are altered.  The default is the
// [Handler], if it implements [Updater].
func WithUpdater(updater Updater) WithUpdaterOption {
	return WithUpdaterOption{
		updater: updater,
	}
}

// WithDonerOption is an [Option] implementation that explicitly
// sets the [Doner] to use.
type WithDonerOption struct {
	doner Doner
}

// apply applies an option.
func (o WithDonerOption) apply(opts *options) {
	opts.doner = o.doner
}

// WithDoner returns an [Option] that can be passed to [Depaginate]
// which sets an [Doner] to be called once all pages are retrieved.
// The default is the [Handler], if it implements [Doner].
func WithDoner(doner Doner) WithDonerOption {
	return WithDonerOption{
		doner: doner,
	}
}

// WithRequestOption is an [Option] implementation that sets the
// initial request.
type WithRequestOption struct {
	req any
}

// apply applies an option.
func (o WithRequestOption) apply(opts *options) {
	opts.initReq = o.req
}

// WithRequest returns an [Option] which sets the request object for
// the initial page load.  By default, the request will be set to nil.
func WithRequest(req any) WithRequestOption {
	return WithRequestOption{
		req: req,
	}
}

// update describes an update to be processed by the [Depaginator]'s
// daemon.  The daemon processes updates to metadata, such as the
// total number of items, as well as issuing new page requests.
type update[T any] interface {
	// applyUpdate applies an update.
	applyUpdate(depag *Depaginator[T])
}

// cancelerFor is an [update] implementation that registers a canceler
// for a specific page.
type cancelerFor[T any] struct {
	page     int                // Index of the page
	cancelFn context.CancelFunc // Function to call to cancel page load
}

// applyUpdate applies an update.
func (u cancelerFor[T]) applyUpdate(depag *Depaginator[T]) {
	depag.cancelers[u.page] = u.cancelFn
}

// withdrawCancelerUpdate is an [update] that withdraws a canceler for
// a specific page.
type withdrawCanceler[T any] int

// applyUpdate applies an update.
func (u withdrawCanceler[T]) applyUpdate(depag *Depaginator[T]) {
	delete(depag.cancelers, int(u))
}

// errorSaver is an [update] implementation that saves an error.
type errorSaver[T any] struct {
	req PageRequest // The request that caused the error
	err error       // The error that was caused
}

// applyUpdate applies an update.
func (u errorSaver[T]) applyUpdate(depag *Depaginator[T]) {
	// Skip context-related errors
	if errors.Is(u.err, context.Canceled) || errors.Is(u.err, context.DeadlineExceeded) {
		return
	}

	// Save the error
	depag.errors = append(depag.errors, PageError{
		PageRequest: u.req,
		Err:         u.err,
	})
}

// itemHandler is an [update] implementation that handles a page of
// items.  The items are handled in a separate goroutine.
type itemHandler[T any] struct {
	idx  int // Page index
	page []T // The page of items to handle
}

// applyUpdate applies an update.
func (u itemHandler[T]) applyUpdate(depag *Depaginator[T]) {
	// Is this page short?
	if len(u.page) < depag.perPage {
		// Got the page count and item count now
		totPages := u.idx + 1
		totItems := depag.perPage*u.idx + len(u.page)
		if depag.totalPages == 0 || depag.totalPages > totPages {
			depag.totalPages = totPages
		}
		if depag.totalItems == 0 || depag.totalItems > totItems {
			depag.totalItems = totItems
		}

		// Cancel pages we no longer need
		for page, canceler := range depag.cancelers {
			if page > u.idx {
				canceler()
			}
		}
	}

	// Compute the base item index and handle the items
	depag.wg.Add(1)
	go u.handle(depag, depag.perPage*u.idx)
}

// handle handles each item in the page.
func (u itemHandler[T]) handle(depag *Depaginator[T], itemBase int) {
	defer depag.wg.Done()

	for i, item := range u.page {
		depag.handler.Handle(depag.ctx, itemBase+i, item)
	}
}

// pageDone is a sentinel [update] implementation that decrements the
// wait group.
type pageDone[T any] struct{}

// applyUpdate applies an update.
func (u pageDone[T]) applyUpdate(depag *Depaginator[T]) {
	depag.wg.Done()
}

// totalItems is an [update] that updates the total number of items to
// expect.
type totalItems[T any] int

// applyUpdate applies an update.
func (u totalItems[T]) applyUpdate(depag *Depaginator[T]) {
	if int(u) > 0 {
		depag.totalItems = int(u)
	}
}

// totalPages is an [update] that updates the total number of pages to
// expect.
type totalPages[T any] int

// applyUpdate applies an update.
func (u totalPages[T]) applyUpdate(depag *Depaginator[T]) {
	if int(u) > 0 {
		depag.totalPages = int(u)
	}
}

// perPage is an [update] that updates the number of items to expect
// in each page.
type perPage[T any] int

// applyUpdate applies an update.
func (u perPage[T]) applyUpdate(depag *Depaginator[T]) {
	if int(u) > 0 {
		depag.perPage = int(u)
	}
}

// bundle is an [update] that bundles together several updates.
type bundle[T any] []update[T]

// applyUpdate applies an update.
func (u bundle[T]) applyUpdate(depag *Depaginator[T]) {
	for _, update := range u {
		update.applyUpdate(depag)
	}
}

// pageRequest is an [update] implementation that requests a page.
type pageRequest[T any] struct {
	idx int // Page index
	req any // Request-specific data
}

// applyUpdate applies an update.
func (u pageRequest[T]) applyUpdate(depag *Depaginator[T]) {
	// Does the page exist?
	if depag.totalPages > 0 && u.idx >= depag.totalPages {
		return
	}

	// Has the page been requested already?
	if depag.pages.CheckAndSet(u.idx) {
		return
	}

	// Place the request
	depag.wg.Add(1)
	go depag.getPage(PageRequest{
		PageIndex: u.idx,
		Request:   u.req,
	})
}
