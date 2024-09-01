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

import "context"

// State describes the state of depagination.  It provides the
// feedback mechanism for requesting updates to the depaginator state,
// such as updating the total number of items or making additional
// page requests.
type State interface {
	// Update allows updating the total number of items, total number
	// of pages, or the items per page.  The arguments passed to
	// Update should be [TotalItems], [TotalPages], or [PerPage]; any
	// other argument types will be ignored.
	Update(updates ...any)

	// Request requests the [Depaginator] retrieve a page.  Note that
	// the page index is 0-based; the first page always has index 0.
	// The request is optional, and can contain any page-specific
	// data, such as a page link.  Duplicate page requests are
	// ignored, as is any request with an index greater than the total
	// number of pages (if known).
	Request(idx int, req any)

	// PerPage retrieves the configured "per page" value for
	// [Depaginator].  This allows a consumer to set the number of
	// items per page when calling [Depaginate] (using the [PerPage]
	// option).  Applications should be careful to not mix this
	// functionality with dynamic collection of the "per page" value,
	// as the value is not protected by any mutex; if using this
	// method, avoid passing [PerPage] to [Depaginator.Update] and
	// arrange for a reasonable default if [PerPage] is not passed to
	// [Depaginate] (in which case, this method will return 0).
	PerPage() int
}

// PageGetter is an interface for a GetPage method that retrieves a
// page specified by the given [PageRequest].  It returns a slice of
// some type or an error.  It may also call methods on the
// [Depaginator] object to submit metadata information, such as
// additional requests to make.
type PageGetter[T any] interface {
	// GetPage is a page retriever function.  It is passed the
	// [Depaginator] object and a [PageRequest] object describing the
	// page to request, and returns a list of items of the appropriate
	// type, or an error.  Methods of [Depaginator] should be called
	// to update data on the maximum number of items, maximum number
	// of pages, items per page, or additional pages to request.  Note
	// that page requests for page indexes that are greater than the
	// maximum known number of pages will be ignored.
	GetPage(ctx context.Context, depag State, req PageRequest) ([]T, error)
}

// PageGetterFunc is a wrapper for a function matching the
// [PageGetter.GetPage] signature.  The wrapper implements the
// [PageGetter] interface, allowing a function to be passed instead of
// an interface implementation.
type PageGetterFunc[T any] func(ctx context.Context, depag State, req PageRequest) ([]T, error)

// GetPage is a page retriever function.  It is passed the
// [Depaginator] object and a [PageRequest] object describing the page
// to request, and returns a list of items of the appropriate type, or
// an error.  Methods of [Depaginator] should be called to update data
// on the maximum number of items, maximum number of pages, items per
// page, or additional pages to request.  Note that page requests for
// page indexes that are greater than the maximum known number of
// pages will be ignored.
func (f PageGetterFunc[T]) GetPage(ctx context.Context, depag State, req PageRequest) ([]T, error) {
	return f(ctx, depag, req)
}

// Handler is an interface for handling items iterated over in a given
// page.  Note that the handler is called from a common goroutine, so
// if extensive processing will be performed, a new goroutine should
// be started from the Handle method.
type Handler[T any] interface {
	// Handle is called for each item in a page of items retrieved by
	// the [PageGetter].  It is called with the item index and the
	// item.
	Handle(ctx context.Context, idx int, item T)
}

// HandlerFunc is a wrapper for a function matching the
// [Handler.Handle] signature.  The wrapper implements the [Handler]
// interface, allowing a function to be passed instead of an interface
// implementation.
type HandlerFunc[T any] func(ctx context.Context, idx int, item T)

// Handle is called for each item in a page of items retrieved by the
// [PageGetter].  It is called with the item index and the item.
func (f HandlerFunc[T]) Handle(ctx context.Context, idx int, item T) {
	f(ctx, idx, item)
}

// Starter is an interface that can be additionally implemented by
// [Handler] implementations.  The Start method will be called before
// [Depaginate] begins its work, allowing the [Handler] to implement
// any initialization it requires.
type Starter interface {
	// Start is called with the initial values of total items, total
	// pages, and items per page.  It should perform any
	// initialization that may be required.
	Start(ctx context.Context, totalItems, totalPages, perPage int)
}

// StarterFunc is a wrapper for a function matching the
// [Starter.Start] signature.  The wrapper implements the [Starter]
// interface, allowing a function to be passed instead of an interface
// implementation.
type StarterFunc func(ctx context.Context, totalItems, totalPages, perPage int)

// Start is called with the initial values of total items, total
// pages, and items per page.  It should perform any initialization
// that may be required.
func (f StarterFunc) Start(ctx context.Context, totalItems, totalPages, perPage int) {
	f(ctx, totalItems, totalPages, perPage)
}

// Updater is an interface that can be additionally implemented by
// [Handler] implementations.  The Update method will be called with
// the updated values of the total items, total pages, and items per
// page, every time the [PageGetter.GetPage] method makes appropriate
// calls to update these values.
type Updater interface {
	// Update is called with the new values of total items, total
	// pages, and items per page.  It should not undertake extensive
	// processing.
	Update(ctx context.Context, totalItems, totalPages, perPage int)
}

// UpdaterFunc is a wrapper for a function matching the
// [Updater.Update] signature.  The wrapper implements the [Updater]
// interface, allowing a function to be passed instead of an interface
// implementation.
type UpdaterFunc func(ctx context.Context, totalItems, totalPages, perPage int)

// Update is called with the new values of total items, total pages,
// and items per page.  It should not undertake extensive processing.
func (f UpdaterFunc) Update(ctx context.Context, totalItems, totalPages, perPage int) {
	f(ctx, totalItems, totalPages, perPage)
}

// Doner is an interface that can be additionally implemented by
// [Handle] implementations.  The Done method will be called once all
// pages have been retrieved and all items have been handled.
type Doner interface {
	// Done is called with the most up-to-date values of total items,
	// total pages, and items per page.  It is called once all pages
	// have been retrieved and all items handled.
	Done(ctx context.Context, totalItems, totalPages, perPage int)
}

// DonerFunc is a wrapper for a function matching the
// [Doner.Done] signature.  The wrapper implements the [Doner]
// interface, allowing a function to be passed instead of an interface
// implementation.
type DonerFunc func(ctx context.Context, totalItems, totalPages, perPage int)

// Done is called with the most up-to-date values of total items,
// total pages, and items per page.  It is called once all pages have
// been retrieved and all items handled.
func (f DonerFunc) Done(ctx context.Context, totalItems, totalPages, perPage int) {
	f(ctx, totalItems, totalPages, perPage)
}
