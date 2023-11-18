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

// PageError contains an error returned by the [API.GetPage] callback,
// along with the failing page request.
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
