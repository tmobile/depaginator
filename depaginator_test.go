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

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func To[T any](v any) T {
	if v == nil {
		var empty T
		return empty
	}

	return v.(T)
}

type mockAPI struct {
	mock.Mock
}

func (m *mockAPI) GetPage(ctx context.Context, pm *PageMeta, req PageRequest) ([]string, error) {
	args := m.Called(ctx, pm, req)

	return To[[]string](args.Get(0)), args.Error(1)
}

func (m *mockAPI) HandleItem(ctx context.Context, idx int, item string) {
	m.Called(ctx, idx, item)
}

func (m *mockAPI) Done(pm PageMeta) {
	m.Called(pm)
}

func TestDepaginate(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page0 := []string{"item 0", "item 1"}
	page1 := []string{"item 2", "item 3"}
	page2 := []string{"item 4"}
	req0 := PageRequest{
		PageIndex: 0,
		Request:   "page 0",
	}
	req1 := PageRequest{
		PageIndex: 1,
		Request:   "page 1",
	}
	req2 := PageRequest{
		PageIndex: 2,
		Request:   "page 2",
	}
	api.On("GetPage", mock.Anything, mock.Anything, req0).Return(page0, nil).Run(func(args mock.Arguments) {
		meta := args[1].(*PageMeta)
		meta.SetPerPage(2)
		meta.AddRequest(req1)
		meta.AddRequest(req2)
	})
	api.On("GetPage", mock.Anything, mock.Anything, req1).Return(page1, nil).Run(func(args mock.Arguments) {
		meta := args[1].(*PageMeta)
		meta.AddRequest(req2)
	})
	api.On("GetPage", mock.Anything, mock.Anything, req2).Once().Return(page2, nil)
	api.On("HandleItem", mock.Anything, 0, "item 0")
	api.On("HandleItem", mock.Anything, 1, "item 1")
	api.On("HandleItem", mock.Anything, 2, "item 2")
	api.On("HandleItem", mock.Anything, 3, "item 3")
	api.On("HandleItem", mock.Anything, 4, "item 4")

	obj := Depaginate[string](ctx, api, req0)
	obj.wg.Wait()

	assert.Nil(t, obj.errors)
	api.AssertExpectations(t)
}

func TestDepaginatorWait(t *testing.T) {
	api := &mockAPI{}
	api.On("Done", PageMeta{PerPage: 5})
	errs := []error{
		PageError{
			PageRequest: PageRequest{PageIndex: 1},
		},
		PageError{
			Err: assert.AnError,
		},
	}
	obj := &Depaginator[string]{
		errors: errs,
		api:    api,
		meta:   &PageMeta{PerPage: 5},
		wg:     &sync.WaitGroup{},
	}

	result := obj.Wait()

	assert.Equal(t, errors.Join(errs...), result)
	api.AssertExpectations(t)
}

func TestDepaginatorRegisterCanceler(t *testing.T) {
	obj := &Depaginator[string]{
		meta: &PageMeta{
			ItemCount: 50,
			PageCount: 10,
			PerPage:   5,
		},
		cancelers: map[int]context.CancelFunc{},
	}

	result := obj.registerCanceler(17, func() {})

	assert.Equal(t, PageMeta{
		ItemCount: 50,
		PageCount: 10,
		PerPage:   5,
	}, result)
	assert.Contains(t, obj.cancelers, 17)
}

type testCanceler struct {
	Canceled bool
}

func (tc *testCanceler) Cancel() {
	tc.Canceled = true
}

func TestDepaginatorCancelPages(t *testing.T) {
	page1 := &testCanceler{}
	page3 := &testCanceler{}
	page5 := &testCanceler{}
	page7 := &testCanceler{}
	obj := &Depaginator[string]{
		cancelers: map[int]context.CancelFunc{
			1: page1.Cancel,
			3: page3.Cancel,
			5: page5.Cancel,
			7: page7.Cancel,
		},
	}

	obj.cancelPages(3)

	assert.False(t, page1.Canceled)
	assert.False(t, page3.Canceled)
	assert.True(t, page5.Canceled)
	assert.True(t, page7.Canceled)
}

func TestDepaginatorIssueRequestsBase(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page := []string{"item 0", "item 1", "item 2", "item 3", "item 4"}
	reqs := []PageRequest{
		{
			PageIndex: 0,
			Request:   "page 0",
		},
		{
			PageIndex: 1,
			Request:   "page 1",
		},
	}
	obj := &Depaginator[string]{
		meta: &PageMeta{
			PerPage: 5,
		},
		api:       api,
		cancelers: map[int]context.CancelFunc{},
		pages:     &pageMap{},
		wg:        &sync.WaitGroup{},
	}
	obj.pages.CheckAndSet(0)
	api.On("GetPage", mock.Anything, mock.Anything, reqs[1]).Return(page, nil)
	api.On("HandleItem", mock.Anything, 5, "item 0")
	api.On("HandleItem", mock.Anything, 6, "item 1")
	api.On("HandleItem", mock.Anything, 7, "item 2")
	api.On("HandleItem", mock.Anything, 8, "item 3")
	api.On("HandleItem", mock.Anything, 9, "item 4")

	obj.issueRequests(ctx, reqs)
	obj.wg.Wait()

	assert.Nil(t, obj.errors)
	api.AssertExpectations(t)
}

func TestDepaginatorIssueRequestsTooManyPages(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page := []string{"item 0", "item 1", "item 2", "item 3", "item 4"}
	reqs := []PageRequest{
		{
			PageIndex: 0,
			Request:   "page 0",
		},
		{
			PageIndex: 1,
			Request:   "page 1",
		},
		{
			PageIndex: 2,
			Request:   "page 2",
		},
	}
	obj := &Depaginator[string]{
		meta: &PageMeta{
			PageCount: 2,
			PerPage:   5,
		},
		api:       api,
		cancelers: map[int]context.CancelFunc{},
		pages:     &pageMap{},
		wg:        &sync.WaitGroup{},
	}
	obj.pages.CheckAndSet(0)
	api.On("GetPage", mock.Anything, mock.Anything, reqs[1]).Return(page, nil)
	api.On("HandleItem", mock.Anything, 5, "item 0")
	api.On("HandleItem", mock.Anything, 6, "item 1")
	api.On("HandleItem", mock.Anything, 7, "item 2")
	api.On("HandleItem", mock.Anything, 8, "item 3")
	api.On("HandleItem", mock.Anything, 9, "item 4")

	obj.issueRequests(ctx, reqs)
	obj.wg.Wait()

	assert.Nil(t, obj.errors)
	api.AssertExpectations(t)
}

func TestDepaginatorPageErrorBase(t *testing.T) {
	req := PageRequest{
		PageIndex: 5,
	}
	obj := &Depaginator[string]{}

	obj.pageError(req, assert.AnError)

	assert.Equal(t, &Depaginator[string]{
		errors: []error{
			PageError{
				PageRequest: req,
				Err:         assert.AnError,
			},
		},
	}, obj)
}

func TestDepaginatorPageErrorCanceled(t *testing.T) {
	req := PageRequest{
		PageIndex: 5,
	}
	obj := &Depaginator[string]{}

	obj.pageError(req, context.Canceled)

	assert.Equal(t, &Depaginator[string]{}, obj)
}

func TestDepaginatorPageErrorDeadlineExceeded(t *testing.T) {
	req := PageRequest{
		PageIndex: 5,
	}
	obj := &Depaginator[string]{}

	obj.pageError(req, context.DeadlineExceeded)

	assert.Equal(t, &Depaginator[string]{}, obj)
}

func TestDepaginatorGetPageBase(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page := []string{"item 0", "item 1", "item 2", "item 3", "item 4"}
	req := PageRequest{
		PageIndex: 1,
		Request:   "page 1",
	}
	page2 := &testCanceler{}
	obj := &Depaginator[string]{
		meta: &PageMeta{
			PageCount: 2,
		},
		api: api,
		cancelers: map[int]context.CancelFunc{
			2: page2.Cancel,
		},
		pages: &pageMap{},
		wg:    &sync.WaitGroup{},
	}
	api.On("GetPage", mock.Anything, &PageMeta{
		PageCount: 2,
	}, req).Return(page, nil).Run(func(args mock.Arguments) {
		meta := args[1].(*PageMeta)
		meta.SetPerPage(5)
	})
	api.On("HandleItem", mock.Anything, 5, "item 0")
	api.On("HandleItem", mock.Anything, 6, "item 1")
	api.On("HandleItem", mock.Anything, 7, "item 2")
	api.On("HandleItem", mock.Anything, 8, "item 3")
	api.On("HandleItem", mock.Anything, 9, "item 4")
	obj.wg.Add(1)

	obj.getPage(ctx, req)
	obj.wg.Wait()

	assert.Nil(t, obj.errors)
	assert.Equal(t, 2, obj.meta.PageCount)
	assert.Equal(t, 10, obj.meta.ItemCount)
	assert.Equal(t, 5, obj.meta.PerPage)
	assert.True(t, page2.Canceled)
	api.AssertExpectations(t)
}

func TestDepaginatorGetPageShortPage(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page := []string{"item 0", "item 1", "item 2", "item 3"}
	req := PageRequest{
		PageIndex: 1,
		Request:   "page 1",
	}
	page2 := &testCanceler{}
	obj := &Depaginator[string]{
		meta: &PageMeta{
			PageCount: 2,
		},
		api: api,
		cancelers: map[int]context.CancelFunc{
			2: page2.Cancel,
		},
		pages: &pageMap{},
		wg:    &sync.WaitGroup{},
	}
	api.On("GetPage", mock.Anything, &PageMeta{
		PageCount: 2,
	}, req).Return(page, nil).Run(func(args mock.Arguments) {
		meta := args[1].(*PageMeta)
		meta.SetPerPage(5)
		meta.AddRequest(PageRequest{
			PageIndex: 2,
			Request:   "page 2",
		})
	})
	api.On("HandleItem", mock.Anything, 5, "item 0")
	api.On("HandleItem", mock.Anything, 6, "item 1")
	api.On("HandleItem", mock.Anything, 7, "item 2")
	api.On("HandleItem", mock.Anything, 8, "item 3")
	obj.wg.Add(1)

	obj.getPage(ctx, req)
	obj.wg.Wait()

	assert.Nil(t, obj.errors)
	assert.Equal(t, 2, obj.meta.PageCount)
	assert.Equal(t, 9, obj.meta.ItemCount)
	assert.Equal(t, 5, obj.meta.PerPage)
	assert.True(t, page2.Canceled)
	api.AssertExpectations(t)
}

func TestDepaginatorGetPageError(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	req := PageRequest{
		PageIndex: 1,
		Request:   "page 1",
	}
	page2 := &testCanceler{}
	obj := &Depaginator[string]{
		meta: &PageMeta{
			PageCount: 2,
		},
		api: api,
		cancelers: map[int]context.CancelFunc{
			2: page2.Cancel,
		},
		pages: &pageMap{},
		wg:    &sync.WaitGroup{},
	}
	api.On("GetPage", mock.Anything, &PageMeta{
		PageCount: 2,
	}, req).Return(nil, assert.AnError)
	obj.wg.Add(1)

	obj.getPage(ctx, req)
	obj.wg.Wait()

	assert.Equal(t, []error{
		PageError{
			PageRequest: req,
			Err:         assert.AnError,
		},
	}, obj.errors)
	assert.Equal(t, 2, obj.meta.PageCount)
	assert.Equal(t, 0, obj.meta.ItemCount)
	assert.Equal(t, 0, obj.meta.PerPage)
	assert.False(t, page2.Canceled)
	api.AssertExpectations(t)
}

func TestDepaginatorGetPageRecurse(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page1 := []string{"item 0", "item 1", "item 2", "item 3", "item 4"}
	page2 := []string{"item 5"}
	req1 := PageRequest{
		PageIndex: 1,
		Request:   "page 1",
	}
	req2 := PageRequest{
		PageIndex: 2,
		Request:   "page 2",
	}
	obj := &Depaginator[string]{
		meta: &PageMeta{
			PageCount: 3,
		},
		api:       api,
		cancelers: map[int]context.CancelFunc{},
		pages:     &pageMap{},
		wg:        &sync.WaitGroup{},
	}
	api.On("GetPage", mock.Anything, &PageMeta{
		PageCount: 3,
	}, req1).Return(page1, nil).Run(func(args mock.Arguments) {
		meta := args[1].(*PageMeta)
		meta.SetPerPage(5)
		meta.AddRequest(req2)
	})
	api.On("GetPage", mock.Anything, &PageMeta{
		PageCount: 3,
		PerPage:   5,
	}, req2).Return(page2, nil)
	api.On("HandleItem", mock.Anything, 5, "item 0")
	api.On("HandleItem", mock.Anything, 6, "item 1")
	api.On("HandleItem", mock.Anything, 7, "item 2")
	api.On("HandleItem", mock.Anything, 8, "item 3")
	api.On("HandleItem", mock.Anything, 9, "item 4")
	api.On("HandleItem", mock.Anything, 10, "item 5")
	obj.wg.Add(1)

	obj.getPage(ctx, req1)
	obj.wg.Wait()

	assert.Nil(t, obj.errors)
	assert.Equal(t, 3, obj.meta.PageCount)
	assert.Equal(t, 11, obj.meta.ItemCount)
	assert.Equal(t, 5, obj.meta.PerPage)
	api.AssertExpectations(t)
}
