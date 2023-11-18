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

import "math/bits"

// pageMap is a bitmap used to represent which pages have been
// handled.  This is a deduplication method used to ensure we don't
// get the same page twice.
type pageMap struct {
	bits []uint // The container for the bits
}

// CheckAndSet checks to see if the specific page is set.  It returns
// true if it is.  Either way, it sets the bit for the specific page.
func (pm *pageMap) CheckAndSet(page int) (result bool) {
	idx, bit := bits.Div(0, uint(page), bits.UintSize)
	if idx >= uint(len(pm.bits)) {
		newMap := make([]uint, idx+1)
		copy(newMap, pm.bits)
		pm.bits = newMap
	} else {
		result = pm.bits[idx]&(1<<bit) != 0
	}

	pm.bits[idx] |= 1 << bit

	return
}
