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

// Page contains the items in one page of the paginated response.
type Page interface {
	// Len returns the total number of items in the page.
	Len() int

	// Get retrieves the requested item.  It should return nil if the
	// requested index is greater than the length of the array, or if
	// it is negative.
	Get(idx int) interface{}
}

// PageRequest describes a request for a specific page.  Most of the
// page request lives in the Request field, which can be anything
// needed by the application; the only other field is the PageIndex
// field, which identifies the index of the page.
type PageRequest struct {
	PageIndex int         // The index of the page
	Request   interface{} // The actual data needed to request the page
}

// PageMeta contains metadata about the page.  This includes such
// information as the number of items per page, the number of pages,
// and the number of items.
type PageMeta struct {
	ItemCount int           // Total number of items
	PageCount int           // Total number of pages
	PerPage   int           // Number if items per page
	Requests  []PageRequest // Requests for subsequent pages
}

// SetItemCount sets the total item count in the PageMeta structure.
// Use this instead of directly setting the value to allow the
// Depaginator to detect the change.
func (pm *PageMeta) SetItemCount(cnt int) {
	if pm.ItemCount == cnt {
		return
	}
	pm.ItemCount = cnt
}

// SetPageCount sets the total page count in the PageMeta structure.
// Use this instead of directly setting the value to allow the
// Depaginator to detect the change.
func (pm *PageMeta) SetPageCount(cnt int) {
	if pm.PageCount == cnt {
		return
	}
	pm.PageCount = cnt
}

// SetPerPage sets the count of items per page in the PageMeta
// structure.  Use this instead of directly setting the value to allow
// the Depaginator to detect the change.
func (pm *PageMeta) SetPerPage(cnt int) {
	if pm.PerPage == cnt {
		return
	}
	pm.PerPage = cnt
}

// AddRequest allows adding a page request to the PageMeta structure.
// Requests can be added directly.  This is a convenience method.
func (pm *PageMeta) AddRequest(req PageRequest) {
	pm.Requests = append(pm.Requests, req)
}

// update updates a PageMeta from another PageMeta.
func (pm *PageMeta) update(meta PageMeta) {
	if meta.ItemCount > 0 {
		pm.SetItemCount(meta.ItemCount)
	}
	if meta.PageCount > 0 {
		pm.SetPageCount(meta.PageCount)
	}
	if meta.PerPage > 0 {
		pm.SetPerPage(meta.PerPage)
	}
}
