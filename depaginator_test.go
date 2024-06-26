// Copyright 2021, 2024 T-Mobile USA, Inc.
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
	"github.com/stretchr/testify/require"
)

type mockCancelFn struct {
	mock.Mock
}

func (m *mockCancelFn) Cancel() {
	m.Called()
}

func TestDepaginateBase(t *testing.T) {
	ctx := context.Background()
	pager := &mockPageGetter{}
	pager.On("GetPage", mock.Anything, mock.Anything, PageRequest{
		PageIndex: 0,
		Request:   "zero",
	}).Return([]string{"one", "two", "three"}, nil).Run(func(args mock.Arguments) {
		dp := args[1].(*Depaginator[string])
		dp.Update(TotalPages(3), PerPage(3))
		dp.Request(1, "one")
		dp.Request(2, "two")
		dp.Request(3, "three")
	})
	pager.On("GetPage", mock.Anything, mock.Anything, PageRequest{
		PageIndex: 1,
		Request:   "one",
	}).Return([]string{"four", "five", "six"}, nil)
	pager.On("GetPage", mock.Anything, mock.Anything, PageRequest{
		PageIndex: 2,
		Request:   "two",
	}).Return([]string{"seven", "eight"}, nil)
	handler := &mockHandler{}
	handler.On("Handle", ctx, 0, "one")
	handler.On("Handle", ctx, 1, "two")
	handler.On("Handle", ctx, 2, "three")
	handler.On("Handle", ctx, 3, "four")
	handler.On("Handle", ctx, 4, "five")
	handler.On("Handle", ctx, 5, "six")
	handler.On("Handle", ctx, 6, "seven")
	handler.On("Handle", ctx, 7, "eight")
	o1 := &mockOption{}
	o1.On("apply", mock.Anything).Run(func(args mock.Arguments) {
		dp := args[0].(*options)
		dp.initReq = "zero"
	})
	o2 := &mockOption{}
	o2.On("apply", mock.Anything)

	dp := Depaginate[string](ctx, pager, handler, o1, o2)
	err := dp.Wait()

	assert.NoError(t, err)
	pager.AssertExpectations(t)
	handler.AssertExpectations(t)
	o1.AssertExpectations(t)
	o2.AssertExpectations(t)
}

func TestDepaginateHandlerFull(t *testing.T) {
	ctx := context.Background()
	pager := &mockPageGetter{}
	pager.On("GetPage", mock.Anything, mock.Anything, PageRequest{
		PageIndex: 0,
		Request:   "zero",
	}).Return([]string{"one", "two", "three"}, nil).Run(func(args mock.Arguments) {
		dp := args[1].(*Depaginator[string])
		dp.Update(TotalPages(3), PerPage(3))
		dp.Request(1, "one")
		dp.Request(2, "two")
		dp.Request(3, "three")
	})
	pager.On("GetPage", mock.Anything, mock.Anything, PageRequest{
		PageIndex: 1,
		Request:   "one",
	}).Return([]string{"four", "five", "six"}, nil)
	pager.On("GetPage", mock.Anything, mock.Anything, PageRequest{
		PageIndex: 2,
		Request:   "two",
	}).Return([]string{"seven", "eight"}, nil)
	handler := &mockHandlerFull{}
	handler.On("Start", ctx, 0, 0, 0)
	handler.On("Handle", ctx, 0, "one")
	handler.On("Handle", ctx, 1, "two")
	handler.On("Handle", ctx, 2, "three")
	handler.On("Handle", ctx, 3, "four")
	handler.On("Handle", ctx, 4, "five")
	handler.On("Handle", ctx, 5, "six")
	handler.On("Handle", ctx, 6, "seven")
	handler.On("Handle", ctx, 7, "eight")
	handler.On("Update", ctx, 0, 3, 3)
	handler.On("Update", ctx, 8, 3, 3)
	handler.On("Done", ctx, 8, 3, 3)
	o1 := &mockOption{}
	o1.On("apply", mock.Anything).Run(func(args mock.Arguments) {
		dp := args[0].(*options)
		dp.initReq = "zero"
	})
	o2 := &mockOption{}
	o2.On("apply", mock.Anything)

	dp := Depaginate[string](ctx, pager, handler, o1, o2)
	err := dp.Wait()

	assert.NoError(t, err)
	pager.AssertExpectations(t)
	handler.AssertExpectations(t)
	o1.AssertExpectations(t)
	o2.AssertExpectations(t)
}

func TestDepaginatorDaemonBase(t *testing.T) {
	ctx := context.Background()
	obj := &Depaginator[string]{
		ctx:     ctx,
		updates: make(chan update[string], DefaultCapacity),
		done:    make(chan struct{}),
	}
	u1 := &mockUpdate{}
	u1.On("applyUpdate", obj)
	obj.updates <- u1
	u2 := &mockUpdate{}
	u2.On("applyUpdate", obj).Run(func(args mock.Arguments) {
		depag := args[0].(*Depaginator[string])
		depag.totalItems = 20
	})
	obj.updates <- u2
	u3 := &mockUpdate{}
	u3.On("applyUpdate", obj).Run(func(args mock.Arguments) {
		depag := args[0].(*Depaginator[string])
		depag.totalPages = 4
	})
	obj.updates <- u3
	u4 := &mockUpdate{}
	u4.On("applyUpdate", obj).Run(func(args mock.Arguments) {
		depag := args[0].(*Depaginator[string])
		depag.perPage = 5
	})
	obj.updates <- u4
	u5 := &mockUpdate{}
	u5.On("applyUpdate", obj)
	obj.updates <- u5
	close(obj.updates)

	obj.daemon()

	select {
	case <-obj.done:
	default:
		assert.Fail(t, "daemon failed to close channel")
	}
	u1.AssertExpectations(t)
	u2.AssertExpectations(t)
	u3.AssertExpectations(t)
	u4.AssertExpectations(t)
	u5.AssertExpectations(t)
}

func TestDepaginatorDaemonWithUpdater(t *testing.T) {
	ctx := context.Background()
	updater := &mockUpdater{}
	updater.On("Update", ctx, 20, 0, 0)
	updater.On("Update", ctx, 20, 4, 0)
	updater.On("Update", ctx, 20, 4, 5)
	obj := &Depaginator[string]{
		ctx:     ctx,
		updater: updater,
		updates: make(chan update[string], DefaultCapacity),
		done:    make(chan struct{}),
	}
	u1 := &mockUpdate{}
	u1.On("applyUpdate", obj)
	obj.updates <- u1
	u2 := &mockUpdate{}
	u2.On("applyUpdate", obj).Run(func(args mock.Arguments) {
		depag := args[0].(*Depaginator[string])
		depag.totalItems = 20
	})
	obj.updates <- u2
	u3 := &mockUpdate{}
	u3.On("applyUpdate", obj).Run(func(args mock.Arguments) {
		depag := args[0].(*Depaginator[string])
		depag.totalPages = 4
	})
	obj.updates <- u3
	u4 := &mockUpdate{}
	u4.On("applyUpdate", obj).Run(func(args mock.Arguments) {
		depag := args[0].(*Depaginator[string])
		depag.perPage = 5
	})
	obj.updates <- u4
	u5 := &mockUpdate{}
	u5.On("applyUpdate", obj)
	obj.updates <- u5
	close(obj.updates)

	obj.daemon()

	select {
	case <-obj.done:
	default:
		assert.Fail(t, "daemon failed to close channel")
	}
	u1.AssertExpectations(t)
	u2.AssertExpectations(t)
	u3.AssertExpectations(t)
	u4.AssertExpectations(t)
	u5.AssertExpectations(t)
}

func TestDepaginatorWaitBase(t *testing.T) {
	obj := &Depaginator[string]{
		totalItems: 20,
		totalPages: 4,
		perPage:    5,
		wg:         &sync.WaitGroup{},
		updates:    make(chan update[string]),
		done:       make(chan struct{}),
	}
	close(obj.done)

	err := obj.Wait()

	assert.NoError(t, err)
	select {
	case <-obj.updates:
	default:
		assert.Fail(t, "Wait failed to close updates channel")
	}
}

func TestDepaginatorWaitWithDoner(t *testing.T) {
	ctx := context.Background()
	doner := &mockDoner{}
	doner.On("Done", ctx, 20, 4, 5)
	obj := &Depaginator[string]{
		ctx:        ctx,
		totalItems: 20,
		totalPages: 4,
		perPage:    5,
		doner:      doner,
		wg:         &sync.WaitGroup{},
		updates:    make(chan update[string]),
		done:       make(chan struct{}),
	}
	close(obj.done)

	err := obj.Wait()

	assert.NoError(t, err)
	select {
	case <-obj.updates:
	default:
		assert.Fail(t, "Wait failed to close updates channel")
	}
	doner.AssertExpectations(t)
}

func TestDepaginatorUpdateInternal(t *testing.T) {
	obj := &Depaginator[string]{
		updates: make(chan update[string], DefaultCapacity),
	}
	u := &mockUpdate{}

	obj.update(u)

	close(obj.updates)
	assert.Len(t, obj.updates, 1)
	assert.Same(t, u, <-obj.updates)
}

func TestDepaginatorGetPageBase(t *testing.T) {
	ctx := context.Background()
	pager := &mockPageGetter{}
	obj := &Depaginator[string]{
		ctx:     ctx,
		pager:   pager,
		updates: make(chan update[string], DefaultCapacity),
	}
	req := PageRequest{
		PageIndex: 5,
		Request:   "five",
	}
	pager.On("GetPage", mock.Anything, obj, req).Return([]string{"one", "two", "three"}, nil)

	obj.getPage(req)

	close(obj.updates)
	updates := []update[string]{}
	for u := range obj.updates {
		updates = append(updates, u)
	}
	assert.Len(t, updates, 4)
	require.IsType(t, cancelerFor[string]{}, updates[0])
	assert.Equal(t, 5, updates[0].(cancelerFor[string]).page)
	assert.Equal(t, withdrawCanceler[string](5), updates[1])
	assert.Equal(t, itemHandler[string]{
		idx:  5,
		page: []string{"one", "two", "three"},
	}, updates[2])
	assert.Equal(t, pageDone[string]{}, updates[3])
	pager.AssertExpectations(t)
}

func TestDepaginatorGetPageError(t *testing.T) {
	ctx := context.Background()
	pager := &mockPageGetter{}
	obj := &Depaginator[string]{
		ctx:     ctx,
		pager:   pager,
		updates: make(chan update[string], DefaultCapacity),
	}
	req := PageRequest{
		PageIndex: 5,
		Request:   "five",
	}
	pager.On("GetPage", mock.Anything, obj, req).Return(nil, assert.AnError)

	obj.getPage(req)

	close(obj.updates)
	updates := []update[string]{}
	for u := range obj.updates {
		updates = append(updates, u)
	}
	assert.Len(t, updates, 4)
	require.IsType(t, cancelerFor[string]{}, updates[0])
	assert.Equal(t, 5, updates[0].(cancelerFor[string]).page)
	assert.Equal(t, withdrawCanceler[string](5), updates[1])
	assert.Equal(t, errorSaver[string]{
		req: req,
		err: assert.AnError,
	}, updates[2])
	assert.Equal(t, pageDone[string]{}, updates[3])
	pager.AssertExpectations(t)
}

func TestDepaginatorUpdateBase(t *testing.T) {
	obj := &Depaginator[string]{
		updates: make(chan update[string], DefaultCapacity),
	}

	obj.Update(TotalItems(20), TotalPages(4), PerPage(5))

	select {
	case update := <-obj.updates:
		assert.Equal(t, bundle[string]{
			totalItems[string](20),
			totalPages[string](4),
			perPage[string](5),
		}, update)
	default:
		assert.Fail(t, "Update failed to send update on channel")
	}
	close(obj.updates)
}

func TestDepaginatorUpdateNoUpdates(t *testing.T) {
	obj := &Depaginator[string]{
		updates: make(chan update[string], DefaultCapacity),
	}

	obj.Update(20, 4, 5)

	select {
	case <-obj.updates:
		assert.Fail(t, "Update sent unexpected update on channel")
	default:
	}
	close(obj.updates)
}

func TestDepaginatorRequest(t *testing.T) {
	obj := &Depaginator[string]{
		updates: make(chan update[string], DefaultCapacity),
	}

	obj.Request(3, "three")

	select {
	case update := <-obj.updates:
		assert.Equal(t, pageRequest[string]{
			idx: 3,
			req: "three",
		}, update)
	default:
		assert.Fail(t, "Request failed to send update on channel")
	}
	close(obj.updates)
}

func TestDepaginatorPerPage(t *testing.T) {
	obj := &Depaginator[string]{
		perPage: 50,
	}

	result := obj.PerPage()

	assert.Equal(t, 50, result)
}
