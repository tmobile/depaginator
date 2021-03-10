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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAPI struct {
	mock.Mock
}

func (m *mockAPI) GetPage(ctx context.Context, pm *PageMeta, req PageRequest) (Page, error) {
	args := m.MethodCalled("GetPage", ctx, pm, req)

	if tmp := args.Get(0); tmp != nil {
		return tmp.(Page), args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *mockAPI) HandleItem(ctx context.Context, idx int, item interface{}) {
	m.MethodCalled("HandleItem", ctx, idx, item)
}

func (m *mockAPI) Done(pm PageMeta) {
	m.MethodCalled("Done", pm)
}

func TestDepaginate(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page0 := &mockPage{}
	page1 := &mockPage{}
	page2 := &mockPage{}
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
	page0.On("Len").Return(2)
	page0.On("Get", 0).Return("item 0")
	page0.On("Get", 1).Return("item 1")
	page1.On("Len").Return(2)
	page1.On("Get", 0).Return("item 2")
	page1.On("Get", 1).Return("item 3")
	page2.On("Len").Return(1)
	page2.On("Get", 0).Return("item 4")
	api.On("HandleItem", mock.Anything, 0, "item 0")
	api.On("HandleItem", mock.Anything, 1, "item 1")
	api.On("HandleItem", mock.Anything, 2, "item 2")
	api.On("HandleItem", mock.Anything, 3, "item 3")
	api.On("HandleItem", mock.Anything, 4, "item 4")

	obj := Depaginate(ctx, api, req0)
	obj.wg.Wait()

	assert.Nil(t, obj.errors)
	api.AssertExpectations(t)
	page0.AssertExpectations(t)
	page1.AssertExpectations(t)
	page2.AssertExpectations(t)
}

func TestDepaginatorWait(t *testing.T) {
	api := &mockAPI{}
	api.On("Done", PageMeta{PerPage: 5})
	errs := []PageError{
		{
			PageRequest: PageRequest{PageIndex: 1},
		},
		{
			Err: assert.AnError,
		},
	}
	obj := &Depaginator{
		errors: errs,
		api:    api,
		meta:   &PageMeta{PerPage: 5},
		wg:     &sync.WaitGroup{},
	}

	result := obj.Wait()

	assert.Equal(t, errs, result)
	api.AssertExpectations(t)
}

func TestDepaginatorRegisterCanceler(t *testing.T) {
	obj := &Depaginator{
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
	obj := &Depaginator{
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
	page := &mockPage{}
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
	obj := &Depaginator{
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
	page.On("Len").Return(5)
	page.On("Get", 0).Return("item 0")
	page.On("Get", 1).Return("item 1")
	page.On("Get", 2).Return("item 2")
	page.On("Get", 3).Return("item 3")
	page.On("Get", 4).Return("item 4")
	api.On("HandleItem", mock.Anything, 5, "item 0")
	api.On("HandleItem", mock.Anything, 6, "item 1")
	api.On("HandleItem", mock.Anything, 7, "item 2")
	api.On("HandleItem", mock.Anything, 8, "item 3")
	api.On("HandleItem", mock.Anything, 9, "item 4")

	obj.issueRequests(ctx, reqs)
	obj.wg.Wait()

	assert.Nil(t, obj.errors)
	api.AssertExpectations(t)
	page.AssertExpectations(t)
}

func TestDepaginatorIssueRequestsTooManyPages(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page := &mockPage{}
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
	obj := &Depaginator{
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
	page.On("Len").Return(5)
	page.On("Get", 0).Return("item 0")
	page.On("Get", 1).Return("item 1")
	page.On("Get", 2).Return("item 2")
	page.On("Get", 3).Return("item 3")
	page.On("Get", 4).Return("item 4")
	api.On("HandleItem", mock.Anything, 5, "item 0")
	api.On("HandleItem", mock.Anything, 6, "item 1")
	api.On("HandleItem", mock.Anything, 7, "item 2")
	api.On("HandleItem", mock.Anything, 8, "item 3")
	api.On("HandleItem", mock.Anything, 9, "item 4")

	obj.issueRequests(ctx, reqs)
	obj.wg.Wait()

	assert.Nil(t, obj.errors)
	api.AssertExpectations(t)
	page.AssertExpectations(t)
}

func TestDepaginatorPageErrorBase(t *testing.T) {
	req := PageRequest{
		PageIndex: 5,
	}
	obj := &Depaginator{}

	obj.pageError(req, assert.AnError)

	assert.Equal(t, &Depaginator{
		errors: []PageError{
			{
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
	obj := &Depaginator{}

	obj.pageError(req, context.Canceled)

	assert.Equal(t, &Depaginator{}, obj)
}

func TestDepaginatorPageErrorDeadlineExceeded(t *testing.T) {
	req := PageRequest{
		PageIndex: 5,
	}
	obj := &Depaginator{}

	obj.pageError(req, context.DeadlineExceeded)

	assert.Equal(t, &Depaginator{}, obj)
}

func TestDepaginatorHandle(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	api.On("HandleItem", ctx, 5, "item")
	obj := &Depaginator{
		api: api,
		wg:  &sync.WaitGroup{},
	}
	obj.wg.Add(1)

	obj.handle(ctx, 5, "item")
	obj.wg.Wait()

	api.AssertExpectations(t)
}

func TestDepaginatorGetPageBase(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page := &mockPage{}
	req := PageRequest{
		PageIndex: 1,
		Request:   "page 1",
	}
	page2 := &testCanceler{}
	obj := &Depaginator{
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
	page.On("Len").Return(5)
	page.On("Get", 0).Return("item 0")
	page.On("Get", 1).Return("item 1")
	page.On("Get", 2).Return("item 2")
	page.On("Get", 3).Return("item 3")
	page.On("Get", 4).Return("item 4")
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
	page.AssertExpectations(t)
}

func TestDepaginatorGetPageShortPage(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	page := &mockPage{}
	req := PageRequest{
		PageIndex: 1,
		Request:   "page 1",
	}
	page2 := &testCanceler{}
	obj := &Depaginator{
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
	page.On("Len").Return(4)
	page.On("Get", 0).Return("item 0")
	page.On("Get", 1).Return("item 1")
	page.On("Get", 2).Return("item 2")
	page.On("Get", 3).Return("item 3")
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
	page.AssertExpectations(t)
}

func TestDepaginatorGetPageError(t *testing.T) {
	ctx := context.Background()
	api := &mockAPI{}
	req := PageRequest{
		PageIndex: 1,
		Request:   "page 1",
	}
	page2 := &testCanceler{}
	obj := &Depaginator{
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

	assert.Equal(t, []PageError{
		{
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
	page1 := &mockPage{}
	page2 := &mockPage{}
	req1 := PageRequest{
		PageIndex: 1,
		Request:   "page 1",
	}
	req2 := PageRequest{
		PageIndex: 2,
		Request:   "page 2",
	}
	obj := &Depaginator{
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
	page1.On("Len").Return(5)
	page1.On("Get", 0).Return("item 0")
	page1.On("Get", 1).Return("item 1")
	page1.On("Get", 2).Return("item 2")
	page1.On("Get", 3).Return("item 3")
	page1.On("Get", 4).Return("item 4")
	page2.On("Len").Return(1)
	page2.On("Get", 0).Return("item 5")
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
	page1.AssertExpectations(t)
	page2.AssertExpectations(t)
}
