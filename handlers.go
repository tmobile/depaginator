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
)

// grow is a utility to ensure that an array has at least the
// specified capacity.  If additional capacity is required, it extends
// the array.
func grow[S ~[]E, E any](s S, n int) S {
	// Append additional items to s to bring its capacity up to n
	if n -= len(s); n > 0 {
		s = append(s, make([]E, n)...)
	}
	return s
}

// ListHandler is an implementation of [Handler] that constructs a
// slice containing all the retrieved items in order.  It can be
// passed to [Depaginate] multiple times, with additional items added
// at the end of the list.  Once [ListHandler.Done] is called (which
// is called by [Depaginator.Wait]), the Items field of the object
// will contain the properly ordered list of items retrieved via the
// [PageGetter].  No constructor is necessary, as a pointer to the
// zero value of ListHandler is valid.
type ListHandler[T any] struct {
	Items []T // Final list of items

	offset     int // Offset of starting item
	totalItems int // Total number of items reported by [Depaginator]
	totalPages int // Total number of pages reported by [Depaginator]
	perPage    int // Items per page reported by [Depaginator]

	actions chan action[T] // Actions to process
	done    chan struct{}  // Used to signal the daemon has exited
}

// action submits an action to the daemon goroutine.
func (lh *ListHandler[T]) action(act action[T]) {
	lh.actions <- act
}

// daemon processes actions.  Using [ListHandler.action] and daemon
// together prevents [ListHandler] from needing to use [sync.Mutex].
func (lh *ListHandler[T]) daemon() {
	defer close(lh.done)
	for act := range lh.actions {
		// Apply the action
		act.applyAction(lh)
	}
}

// Start is called with the initial values of total items, total
// pages, and items per page.  It should perform any initialization
// that may be required.
func (lh *ListHandler[T]) Start(_ context.Context, totalItems, totalPages, perPage int) {
	// Initialize the algorithm
	lh.offset = len(lh.Items)
	lh.totalItems = totalItems
	lh.totalPages = totalPages
	lh.perPage = perPage
	lh.actions = make(chan action[T], DefaultCapacity)
	lh.done = make(chan struct{})

	// Check if we can select an initial size for the Items list
	if lh.totalItems > 0 {
		lh.Items = grow(lh.Items, lh.offset+lh.totalItems)
	} else if lh.totalPages > 0 && lh.perPage > 0 {
		lh.Items = grow(lh.Items, lh.offset+lh.totalPages*lh.perPage)
	} else if lh.perPage > 0 {
		lh.Items = grow(lh.Items, lh.offset+lh.perPage)
	}

	// Start the daemon
	go lh.daemon()
}

// Done is called with the most up-to-date values of total items,
// total pages, and items per page.  It is called once all pages have
// been retrieved and all items handled.
func (lh *ListHandler[T]) Done(_ context.Context, totalItems, _, _ int) {
	// Wait for processing to be completed and zero the channels
	close(lh.actions)
	<-lh.done
	lh.actions = nil
	lh.done = nil

	// Resize the slice to include just the items we got; totalItems
	// is guaranteed to be correct at this point
	lh.Items = lh.Items[:lh.offset+totalItems]
}

// Handle is called for each item in a page of items retrieved by the
// [PageGetter].  It is called with the item index and the item.
func (lh *ListHandler[T]) Handle(_ context.Context, idx int, item T) {
	lh.action(handleItem[T]{
		idx:  idx,
		item: item,
	})
}

// Update is called with the new values of total items, total pages,
// and items per page.  It should not undertake extensive processing.
func (lh *ListHandler[T]) Update(_ context.Context, totalItems, totalPages, perPage int) {
	lh.action(listUpdate[T]{
		totalItems: totalItems,
		totalPages: totalPages,
		perPage:    perPage,
	})
}

// action specifies an action to perform on a [ListHandler] instance.
type action[T any] interface {
	// applyAction applies an action.
	applyAction(lh *ListHandler[T])
}

// handleItem is an implementation of [action] that handles an item,
// adding it to the list maintained in [ListHandler] at the correct
// index.
type handleItem[T any] struct {
	idx  int // Index of the item in the list
	item T   // Item to be handled
}

// applyAction applies an action.
func (a handleItem[T]) applyAction(lh *ListHandler[T]) {
	// Do we need to grow the list?
	if lh.offset+a.idx >= len(lh.Items) {
		if lh.perPage > 0 {
			lh.Items = grow(lh.Items, lh.offset+a.idx+lh.perPage)
		} else {
			lh.Items = grow(lh.Items, lh.offset+a.idx+1)
		}
	}

	// Save the item
	lh.Items[lh.offset+a.idx] = a.item
}

// listUpdate is an implementation of [action] that saves updates to
// the total number of items, total number of pages, and items per
// page, as reported by [Depaginate].  It uses this information to
// preallocate the list of items.
type listUpdate[T any] struct {
	totalItems int // Total number of items
	totalPages int // Total number of pages
	perPage    int // Number of items per page
}

// applyAction applies an action.
func (a listUpdate[T]) applyAction(lh *ListHandler[T]) {
	// Save the update
	lh.totalItems = a.totalItems
	lh.totalPages = a.totalPages
	lh.perPage = a.perPage

	// Update the capacity if warranted
	if lh.totalItems > 0 {
		lh.Items = grow(lh.Items, lh.offset+lh.totalItems)
	} else if lh.totalPages > 0 && lh.perPage > 0 {
		lh.Items = grow(lh.Items, lh.offset+lh.totalPages*lh.perPage)
	}
}
