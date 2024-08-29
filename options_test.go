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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockOption struct {
	mock.Mock
}

func (m *mockOption) apply(opts *options) {
	m.Called(opts)
}

func TestTotalItemsImplementsOption(t *testing.T) {
	assert.Implements(t, (*Option)(nil), TotalItems(0))
}

func TestTotalItemsApply(t *testing.T) {
	opts := options{}
	obj := TotalItems(5)

	obj.apply(&opts)

	assert.Equal(t, 5, opts.totalItems)
}

func TestTotalPagesImplementsOption(t *testing.T) {
	assert.Implements(t, (*Option)(nil), TotalPages(0))
}

func TestTotalPagesApply(t *testing.T) {
	opts := options{}
	obj := TotalPages(5)

	obj.apply(&opts)

	assert.Equal(t, 5, opts.totalPages)
}

func TestPerPageImplementsOption(t *testing.T) {
	assert.Implements(t, (*Option)(nil), PerPage(0))
}

func TestPerPageApply(t *testing.T) {
	opts := options{}
	obj := PerPage(5)

	obj.apply(&opts)

	assert.Equal(t, 5, opts.perPage)
}

func TestCapacityImplementsOption(t *testing.T) {
	assert.Implements(t, (*Option)(nil), Capacity(0))
}

func TestCapacityApply(t *testing.T) {
	opts := options{}
	obj := Capacity(5)

	obj.apply(&opts)

	assert.Equal(t, 5, opts.capacity)
}

func TestWithStarterOptionImplementsOption(t *testing.T) {
	assert.Implements(t, (*Option)(nil), WithStarterOption{})
}

func TestWithStarterOptionApply(t *testing.T) {
	starter := &mockStarter{}
	obj := WithStarterOption{
		starter: starter,
	}
	opts := options{}

	obj.apply(&opts)

	assert.Same(t, starter, opts.starter)
}

func TestWithStarter(t *testing.T) {
	starter := &mockStarter{}

	result := WithStarter(starter)

	assert.Equal(t, WithStarterOption{
		starter: starter,
	}, result)
}

func TestWithUpdaterOptionImplementsOption(t *testing.T) {
	assert.Implements(t, (*Option)(nil), WithUpdaterOption{})
}

func TestWithUpdaterOptionApply(t *testing.T) {
	updater := &mockUpdater{}
	obj := WithUpdaterOption{
		updater: updater,
	}
	opts := options{}

	obj.apply(&opts)

	assert.Same(t, updater, opts.updater)
}

func TestWithUpdater(t *testing.T) {
	updater := &mockUpdater{}

	result := WithUpdater(updater)

	assert.Equal(t, WithUpdaterOption{
		updater: updater,
	}, result)
}

func TestWithDonerOptionImplementsOption(t *testing.T) {
	assert.Implements(t, (*Option)(nil), WithDonerOption{})
}

func TestWithDonerOptionApply(t *testing.T) {
	doner := &mockDoner{}
	obj := WithDonerOption{
		doner: doner,
	}
	opts := options{}

	obj.apply(&opts)

	assert.Same(t, doner, opts.doner)
}

func TestWithDoner(t *testing.T) {
	doner := &mockDoner{}

	result := WithDoner(doner)

	assert.Equal(t, WithDonerOption{
		doner: doner,
	}, result)
}

func TestWithRequestOptionImplementsOption(t *testing.T) {
	assert.Implements(t, (*Option)(nil), WithRequestOption{})
}

func TestWithRequestOptionApply(t *testing.T) {
	obj := WithRequestOption{
		req: "request",
	}
	opts := options{}

	obj.apply(&opts)

	assert.Equal(t, "request", opts.initReq)
}

func TestWithRequest(t *testing.T) {
	result := WithRequest("request")

	assert.Equal(t, WithRequestOption{
		req: "request",
	}, result)
}

type mockUpdate struct {
	mock.Mock
}

func (m *mockUpdate) applyUpdate(depag *Depaginator[string]) { //nolint:unused
	m.Called(depag)
}

func TestCancelerForImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), cancelerFor[string]{})
}

func TestCancelerForApplyUpdate(t *testing.T) {
	cancelFn := func() {}
	obj := cancelerFor[string]{
		page:     5,
		cancelFn: cancelFn,
	}
	depag := &Depaginator[string]{
		cancelers: map[int]context.CancelFunc{},
	}

	obj.applyUpdate(depag)

	assert.Contains(t, depag.cancelers, 5)
}

func TestWithdrawCancelerImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), withdrawCanceler[string](0))
}

func TestWithdrawCancelerApplyUpdate(t *testing.T) {
	obj := withdrawCanceler[string](5)
	depag := &Depaginator[string]{
		cancelers: map[int]context.CancelFunc{
			5: nil,
		},
	}

	obj.applyUpdate(depag)

	assert.NotContains(t, depag.cancelers, 5)
}

func TestErrorSaverImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), errorSaver[string]{})
}

func TestErrorSaverApplyUpdateBase(t *testing.T) {
	obj := errorSaver[string]{
		req: PageRequest{
			PageIndex: 5,
		},
		err: assert.AnError,
	}
	depag := &Depaginator[string]{}

	obj.applyUpdate(depag)

	assert.Equal(t, &Depaginator[string]{
		errors: []error{
			PageError{
				PageRequest: PageRequest{
					PageIndex: 5,
				},
				Err: assert.AnError,
			},
		},
	}, depag)
}

func TestErrorSaverApplyUpdateCanceled(t *testing.T) {
	obj := errorSaver[string]{
		req: PageRequest{
			PageIndex: 5,
		},
		err: context.Canceled,
	}
	depag := &Depaginator[string]{}

	obj.applyUpdate(depag)

	assert.Equal(t, &Depaginator[string]{}, depag)
}

func TestErrorSaverApplyUpdateDeadlineExceeded(t *testing.T) {
	obj := errorSaver[string]{
		req: PageRequest{
			PageIndex: 5,
		},
		err: context.DeadlineExceeded,
	}
	depag := &Depaginator[string]{}

	obj.applyUpdate(depag)

	assert.Equal(t, &Depaginator[string]{}, depag)
}

func TestItemHandlerImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), itemHandler[string]{})
}

func TestItemHandlerApplyupdateBase(t *testing.T) {
	ctx := context.Background()
	handler := &mockHandler{}
	handler.On("Handle", ctx, 25, "foo")
	handler.On("Handle", ctx, 26, "bar")
	handler.On("Handle", ctx, 27, "baz")
	cancel4 := &mockCancelFn{}
	cancel6 := &mockCancelFn{}
	cancel6.On("Cancel")
	obj := itemHandler[string]{
		idx:  5,
		page: []string{"foo", "bar", "baz"},
	}
	depag := &Depaginator[string]{
		ctx:     ctx,
		perPage: 5,
		handler: handler,
		cancelers: map[int]context.CancelFunc{
			4: cancel4.Cancel,
			6: cancel6.Cancel,
		},
		wg: &sync.WaitGroup{},
	}

	obj.applyUpdate(depag)

	depag.wg.Wait()
	assert.Equal(t, 6, depag.totalPages)
	assert.Equal(t, 28, depag.totalItems)
	cancel4.AssertExpectations(t)
	cancel6.AssertExpectations(t)
	handler.AssertExpectations(t)
}

func TestItemHandlerApplyupdateMorePages(t *testing.T) {
	ctx := context.Background()
	handler := &mockHandler{}
	handler.On("Handle", ctx, 25, "foo")
	handler.On("Handle", ctx, 26, "bar")
	handler.On("Handle", ctx, 27, "baz")
	handler.On("Handle", ctx, 28, "bink")
	handler.On("Handle", ctx, 29, "qux")
	cancel4 := &mockCancelFn{}
	cancel6 := &mockCancelFn{}
	obj := itemHandler[string]{
		idx:  5,
		page: []string{"foo", "bar", "baz", "bink", "qux"},
	}
	depag := &Depaginator[string]{
		ctx:     ctx,
		perPage: 5,
		handler: handler,
		cancelers: map[int]context.CancelFunc{
			4: cancel4.Cancel,
			6: cancel6.Cancel,
		},
		wg: &sync.WaitGroup{},
	}

	obj.applyUpdate(depag)

	depag.wg.Wait()
	assert.Equal(t, 0, depag.totalPages)
	assert.Equal(t, 0, depag.totalItems)
	cancel4.AssertExpectations(t)
	cancel6.AssertExpectations(t)
	handler.AssertExpectations(t)
}

func TestItemHandlerHandle(t *testing.T) {
	ctx := context.Background()
	handler := &mockHandler{}
	handler.On("Handle", ctx, 25, "foo")
	handler.On("Handle", ctx, 26, "bar")
	handler.On("Handle", ctx, 27, "baz")
	obj := itemHandler[string]{
		idx:  5,
		page: []string{"foo", "bar", "baz"},
	}
	depag := &Depaginator[string]{
		ctx:     ctx,
		handler: handler,
		wg:      &sync.WaitGroup{},
	}
	depag.wg.Add(1)

	obj.handle(depag, 25)

	depag.wg.Wait()
	handler.AssertExpectations(t)
}

func TestPageDoneImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), pageDone[string]{})
}

func TestPageDoneApplyUpdate(_ *testing.T) {
	obj := pageDone[string]{}
	depag := &Depaginator[string]{
		wg: &sync.WaitGroup{},
	}
	depag.wg.Add(1)

	obj.applyUpdate(depag)

	depag.wg.Wait()
	// Passes if the waitgroup doesn't wait
}

func TestTotalItemsImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), totalItems[string](0))
}

func TestTotalItemsApplyUpdateBase(t *testing.T) {
	obj := totalItems[string](5)
	depag := &Depaginator[string]{
		totalItems: 3,
	}

	obj.applyUpdate(depag)

	assert.Equal(t, 5, depag.totalItems)
}

func TestTotalItemsApplyUpdateZero(t *testing.T) {
	obj := totalItems[string](0)
	depag := &Depaginator[string]{
		totalItems: 3,
	}

	obj.applyUpdate(depag)

	assert.Equal(t, 3, depag.totalItems)
}

func TestTotalPagesImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), totalPages[string](0))
}

func TestTotalPagesApplyUpdateBase(t *testing.T) {
	obj := totalPages[string](5)
	depag := &Depaginator[string]{
		totalPages: 3,
	}

	obj.applyUpdate(depag)

	assert.Equal(t, 5, depag.totalPages)
}

func TestTotalPagesApplyUpdateZero(t *testing.T) {
	obj := totalPages[string](0)
	depag := &Depaginator[string]{
		totalPages: 3,
	}

	obj.applyUpdate(depag)

	assert.Equal(t, 3, depag.totalPages)
}

func TestPerPageImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), perPage[string](0))
}

func TestPerPageApplyUpdateBase(t *testing.T) {
	obj := perPage[string](5)
	depag := &Depaginator[string]{
		perPage: 3,
	}

	obj.applyUpdate(depag)

	assert.Equal(t, 5, depag.perPage)
}

func TestPerPageApplyUpdateZero(t *testing.T) {
	obj := perPage[string](0)
	depag := &Depaginator[string]{
		perPage: 3,
	}

	obj.applyUpdate(depag)

	assert.Equal(t, 3, depag.perPage)
}

func TestBundleImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), bundle[string]{})
}

func TestBundleApplyUpdate(t *testing.T) {
	depag := &Depaginator[string]{}
	u1 := &mockUpdate{}
	u1.On("applyUpdate", depag)
	u2 := &mockUpdate{}
	u2.On("applyUpdate", depag)
	u3 := &mockUpdate{}
	u3.On("applyUpdate", depag)
	obj := bundle[string]{u1, u2, u3}

	obj.applyUpdate(depag)

	u1.AssertExpectations(t)
	u2.AssertExpectations(t)
	u3.AssertExpectations(t)
}

func TestPageRequestImplementsUpdate(t *testing.T) {
	assert.Implements(t, (*update[string])(nil), pageRequest[string]{})
}

func TestPageRequestApplyUpdateBase(t *testing.T) {
	ctx := context.Background()
	pager := &mockPageGetter{}
	obj := pageRequest[string]{
		idx: 3,
		req: "three",
	}
	depag := &Depaginator[string]{
		ctx:        ctx,
		totalPages: 5,
		pager:      pager,
		pages:      &pageMap{},
		wg:         &sync.WaitGroup{},
		updates:    make(chan update[string], DefaultCapacity),
	}
	pager.On("GetPage", mock.Anything, depag, PageRequest{
		PageIndex: 3,
		Request:   "three",
	}).Return([]string{"foo", "bar", "baz"}, nil)

	obj.applyUpdate(depag)

	updates := []update[string]{}
	go func() {
		for u := range depag.updates {
			updates = append(updates, u)
			if _, ok := u.(pageDone[string]); ok {
				depag.wg.Done()
			}
		}
	}()
	depag.wg.Wait()
	close(depag.updates)
	assert.Len(t, updates, 4)
	pager.AssertExpectations(t)
}

func TestPageRequestApplyUpdatePageVisited(t *testing.T) {
	pager := &mockPageGetter{}
	obj := pageRequest[string]{
		idx: 3,
		req: "three",
	}
	depag := &Depaginator[string]{
		totalPages: 5,
		pager:      pager,
		pages:      &pageMap{},
		wg:         &sync.WaitGroup{},
		updates:    make(chan update[string], DefaultCapacity),
	}
	depag.pages.CheckAndSet(3)

	obj.applyUpdate(depag)

	depag.wg.Wait()
	close(depag.updates)
	updates := []update[string]{}
	for u := range depag.updates {
		updates = append(updates, u)
	}
	assert.Equal(t, []update[string]{}, updates)
	pager.AssertExpectations(t)
}

func TestPageRequestApplyUpdateNoMorePages(t *testing.T) {
	pager := &mockPageGetter{}
	obj := pageRequest[string]{
		idx: 5,
		req: "five",
	}
	depag := &Depaginator[string]{
		totalPages: 5,
		pager:      pager,
		pages:      &pageMap{},
		wg:         &sync.WaitGroup{},
		updates:    make(chan update[string], DefaultCapacity),
	}

	obj.applyUpdate(depag)

	depag.wg.Wait()
	close(depag.updates)
	updates := []update[string]{}
	for u := range depag.updates {
		updates = append(updates, u)
	}
	assert.Equal(t, []update[string]{}, updates)
	pager.AssertExpectations(t)
}
