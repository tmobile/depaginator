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

package depaginator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockPage struct {
	mock.Mock
}

func (m *mockPage) Len() int {
	args := m.MethodCalled("Len")

	return args.Int(0)
}

func (m *mockPage) Get(idx int) interface{} {
	args := m.MethodCalled("Get", idx)

	return args.Get(0)
}

func TestPageMetaSetItemCountBase(t *testing.T) {
	obj := &PageMeta{}

	obj.SetItemCount(5)

	assert.Equal(t, &PageMeta{
		ItemCount: 5,
	}, obj)
}

func TestPageMetaSetItemCountUnchanged(t *testing.T) {
	obj := &PageMeta{
		ItemCount: 5,
	}

	obj.SetItemCount(5)

	assert.Equal(t, &PageMeta{
		ItemCount: 5,
	}, obj)
}

func TestPageMetaSetPageCountBase(t *testing.T) {
	obj := &PageMeta{}

	obj.SetPageCount(5)

	assert.Equal(t, &PageMeta{
		PageCount: 5,
	}, obj)
}

func TestPageMetaSetPageCountUnchanged(t *testing.T) {
	obj := &PageMeta{
		PageCount: 5,
	}

	obj.SetPageCount(5)

	assert.Equal(t, &PageMeta{
		PageCount: 5,
	}, obj)
}

func TestPageMetaSetPerPageBase(t *testing.T) {
	obj := &PageMeta{}

	obj.SetPerPage(5)

	assert.Equal(t, &PageMeta{
		PerPage: 5,
	}, obj)
}

func TestPageMetaSetPerPageUnchanged(t *testing.T) {
	obj := &PageMeta{
		PerPage: 5,
	}

	obj.SetPerPage(5)

	assert.Equal(t, &PageMeta{
		PerPage: 5,
	}, obj)
}

func TestPageMetaAddRequest(t *testing.T) {
	req := PageRequest{
		PageIndex: 5,
		Request:   "request",
	}
	obj := &PageMeta{}

	obj.AddRequest(req)

	assert.Equal(t, &PageMeta{
		Requests: []PageRequest{req},
	}, obj)
}

func TestPageMetaUpdateBase(t *testing.T) {
	obj := &PageMeta{}
	meta := PageMeta{
		ItemCount: 50,
		PageCount: 10,
		PerPage:   5,
	}

	obj.update(meta)

	assert.Equal(t, &PageMeta{
		ItemCount: 50,
		PageCount: 10,
		PerPage:   5,
	}, obj)
}

func TestPageMetaUpdateUnset(t *testing.T) {
	obj := &PageMeta{}
	meta := PageMeta{}

	obj.update(meta)

	assert.Equal(t, &PageMeta{}, obj)
}
