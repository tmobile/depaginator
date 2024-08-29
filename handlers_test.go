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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGrowBase(t *testing.T) {
	result := grow([]string(nil), 5)

	assert.GreaterOrEqual(t, cap(result), 5)
}

func TestGrowUnneeded(t *testing.T) {
	result := grow(make([]string, 7), 5)

	assert.GreaterOrEqual(t, cap(result), 7)
}

func TestGrowExisting(t *testing.T) {
	result := grow(make([]string, 3), 5)

	assert.GreaterOrEqual(t, cap(result), 5)
}

func TestListHandlerImplementsInterfaces(t *testing.T) {
	assert.Implements(t, (*Handler[string])(nil), &ListHandler[string]{})
	assert.Implements(t, (*Starter)(nil), &ListHandler[string]{})
	assert.Implements(t, (*Updater)(nil), &ListHandler[string]{})
	assert.Implements(t, (*Doner)(nil), &ListHandler[string]{})
}

func TestListHandlerAction(t *testing.T) {
	obj := &ListHandler[string]{
		actions: make(chan action[string], DefaultCapacity),
	}
	act := &mockAction{}

	obj.action(act)

	close(obj.actions)
	assert.Len(t, obj.actions, 1)
	assert.Same(t, act, <-obj.actions)
}

func TestListHandlerDaemon(t *testing.T) {
	obj := &ListHandler[string]{
		actions: make(chan action[string], DefaultCapacity),
		done:    make(chan struct{}),
	}
	act1 := &mockAction{}
	act1.On("applyAction", obj)
	obj.actions <- act1
	act2 := &mockAction{}
	act2.On("applyAction", obj)
	obj.actions <- act2
	close(obj.actions)

	obj.daemon()

	select {
	case <-obj.done:
	default:
		assert.Fail(t, "daemon failed to close channel")
	}
	act1.AssertExpectations(t)
	act2.AssertExpectations(t)
}

func TestListHandlerStartBase(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{}

	obj.Start(ctx, 20, 4, 5)
	close(obj.actions)
	<-obj.done

	assert.Equal(t, 0, obj.offset)
	assert.Equal(t, 20, obj.totalItems)
	assert.Equal(t, 4, obj.totalPages)
	assert.Equal(t, 5, obj.perPage)
	assert.GreaterOrEqual(t, cap(obj.Items), 20)
}

func TestListHandlerStartWithOffsetBase(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{
		Items: []string{"foo", "bar", "baz"},
	}

	obj.Start(ctx, 20, 4, 5)
	close(obj.actions)
	<-obj.done

	assert.Equal(t, 3, obj.offset)
	assert.Equal(t, 20, obj.totalItems)
	assert.Equal(t, 4, obj.totalPages)
	assert.Equal(t, 5, obj.perPage)
	assert.GreaterOrEqual(t, cap(obj.Items), 23)
}

func TestListHandlerStartWithPages(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{}

	obj.Start(ctx, 0, 4, 5)
	close(obj.actions)
	<-obj.done

	assert.Equal(t, 0, obj.offset)
	assert.Equal(t, 0, obj.totalItems)
	assert.Equal(t, 4, obj.totalPages)
	assert.Equal(t, 5, obj.perPage)
	assert.GreaterOrEqual(t, cap(obj.Items), 20)
}

func TestListHandlerStartWithOffsetWithPages(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{
		Items: []string{"foo", "bar", "baz"},
	}

	obj.Start(ctx, 0, 4, 5)
	close(obj.actions)
	<-obj.done

	assert.Equal(t, 3, obj.offset)
	assert.Equal(t, 0, obj.totalItems)
	assert.Equal(t, 4, obj.totalPages)
	assert.Equal(t, 5, obj.perPage)
	assert.GreaterOrEqual(t, cap(obj.Items), 23)
}

func TestListHandlerStartWithPerPage(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{}

	obj.Start(ctx, 0, 0, 5)
	close(obj.actions)
	<-obj.done

	assert.Equal(t, 0, obj.offset)
	assert.Equal(t, 0, obj.totalItems)
	assert.Equal(t, 0, obj.totalPages)
	assert.Equal(t, 5, obj.perPage)
	assert.GreaterOrEqual(t, cap(obj.Items), 5)
}

func TestListHandlerStartWithOffsetWithPerPage(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{
		Items: []string{"foo", "bar", "baz"},
	}

	obj.Start(ctx, 0, 0, 5)
	close(obj.actions)
	<-obj.done

	assert.Equal(t, 3, obj.offset)
	assert.Equal(t, 0, obj.totalItems)
	assert.Equal(t, 0, obj.totalPages)
	assert.Equal(t, 5, obj.perPage)
	assert.GreaterOrEqual(t, cap(obj.Items), 8)
}

func TestListHandlerStartNoData(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{}

	obj.Start(ctx, 0, 0, 0)
	close(obj.actions)
	<-obj.done

	assert.Equal(t, 0, obj.offset)
	assert.Equal(t, 0, obj.totalItems)
	assert.Equal(t, 0, obj.totalPages)
	assert.Equal(t, 0, obj.perPage)
	assert.Nil(t, obj.Items)
}

func TestListHandlerStartWithOffsetNoData(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{
		Items: []string{"foo", "bar", "baz"},
	}

	obj.Start(ctx, 0, 0, 0)
	close(obj.actions)
	<-obj.done

	assert.Equal(t, 3, obj.offset)
	assert.Equal(t, 0, obj.totalItems)
	assert.Equal(t, 0, obj.totalPages)
	assert.Equal(t, 0, obj.perPage)
	assert.Equal(t, []string{"foo", "bar", "baz"}, obj.Items)
}

func TestListHandlerDoneBase(t *testing.T) {
	ctx := context.Background()
	actions := make(chan action[string], DefaultCapacity)
	obj := &ListHandler[string]{
		Items:   []string{"foo", "bar", "baz", "bink", "qux"},
		actions: actions,
		done:    make(chan struct{}),
	}
	close(obj.done)

	obj.Done(ctx, 3, 5, 7)

	assert.Equal(t, []string{"foo", "bar", "baz"}, obj.Items)
	assert.Nil(t, obj.actions)
	assert.Nil(t, obj.done)
	select {
	case <-actions:
	default:
		assert.Fail(t, "Done failed to close actions channel")
	}
}

func TestListHandlerDoneWithOffset(t *testing.T) {
	ctx := context.Background()
	actions := make(chan action[string], DefaultCapacity)
	obj := &ListHandler[string]{
		Items:   []string{"foo", "bar", "baz", "bink", "qux"},
		offset:  1,
		actions: actions,
		done:    make(chan struct{}),
	}
	close(obj.done)

	obj.Done(ctx, 3, 5, 7)

	assert.Equal(t, []string{"foo", "bar", "baz", "bink"}, obj.Items)
	assert.Nil(t, obj.actions)
	assert.Nil(t, obj.done)
	select {
	case <-actions:
	default:
		assert.Fail(t, "Done failed to close actions channel")
	}
}

func TestListHandlerHandle(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{
		actions: make(chan action[string], DefaultCapacity),
	}

	obj.Handle(ctx, 3, "three")

	select {
	case action := <-obj.actions:
		assert.Equal(t, handleItem[string]{
			idx:  3,
			item: "three",
		}, action)
	default:
		assert.Fail(t, "Handle failed to send action on channel")
	}
	close(obj.actions)
}

func TestListHandlerUpdate(t *testing.T) {
	ctx := context.Background()
	obj := &ListHandler[string]{
		actions: make(chan action[string], DefaultCapacity),
	}

	obj.Update(ctx, 20, 4, 5)

	select {
	case action := <-obj.actions:
		assert.Equal(t, listUpdate[string]{
			totalItems: 20,
			totalPages: 4,
			perPage:    5,
		}, action)
	default:
		assert.Fail(t, "Update failed to send action on channel")
	}
	close(obj.actions)
}

type mockAction struct {
	mock.Mock
}

func (m *mockAction) applyAction(lh *ListHandler[string]) { //nolint:unused
	m.Called(lh)
}

func TestHandleItemImplementsAction(t *testing.T) {
	assert.Implements(t, (*action[string])(nil), handleItem[string]{})
}

func TestHandleItemApplyActionBase(t *testing.T) {
	obj := handleItem[string]{
		idx:  3,
		item: "three",
	}
	lh := &ListHandler[string]{
		Items: make([]string, 5),
	}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 5)
	assert.Equal(t, "three", lh.Items[3])
}

func TestHandleItemApplyActionWithOffset(t *testing.T) {
	obj := handleItem[string]{
		idx:  3,
		item: "three",
	}
	lh := &ListHandler[string]{
		Items:  make([]string, 5),
		offset: 1,
	}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 5)
	assert.Equal(t, "three", lh.Items[4])
}

func TestHandleItemApplyActionGrowBase(t *testing.T) {
	obj := handleItem[string]{
		idx:  3,
		item: "three",
	}
	lh := &ListHandler[string]{}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 3)
	assert.Equal(t, "three", lh.Items[3])
}

func TestHandleItemApplyActionGrowBaseWithOffset(t *testing.T) {
	obj := handleItem[string]{
		idx:  3,
		item: "three",
	}
	lh := &ListHandler[string]{
		offset: 1,
	}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 4)
	assert.Equal(t, "three", lh.Items[4])
}

func TestHandleItemApplyActionGrowPerPage(t *testing.T) {
	obj := handleItem[string]{
		idx:  3,
		item: "three",
	}
	lh := &ListHandler[string]{
		perPage: 5,
	}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 5)
	assert.Equal(t, "three", lh.Items[3])
}

func TestHandleItemApplyActionGrowPerPageWithOffset(t *testing.T) {
	obj := handleItem[string]{
		idx:  3,
		item: "three",
	}
	lh := &ListHandler[string]{
		offset:  1,
		perPage: 5,
	}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 5)
	assert.Equal(t, "three", lh.Items[4])
}

func TestListUpdateImplementsAction(t *testing.T) {
	assert.Implements(t, (*action[string])(nil), listUpdate[string]{})
}

func TestListUpdateApplyActionBase(t *testing.T) {
	obj := listUpdate[string]{
		totalItems: 20,
		totalPages: 4,
		perPage:    5,
	}
	lh := &ListHandler[string]{}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 20)
	assert.Equal(t, 20, lh.totalItems)
	assert.Equal(t, 4, lh.totalPages)
	assert.Equal(t, 5, lh.perPage)
}

func TestListUpdateApplyActionWithOffset(t *testing.T) {
	obj := listUpdate[string]{
		totalItems: 20,
		totalPages: 4,
		perPage:    5,
	}
	lh := &ListHandler[string]{
		offset: 1,
	}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 21)
	assert.Equal(t, 20, lh.totalItems)
	assert.Equal(t, 4, lh.totalPages)
	assert.Equal(t, 5, lh.perPage)
}

func TestListUpdateApplyActionNoTotal(t *testing.T) {
	obj := listUpdate[string]{
		totalPages: 4,
		perPage:    5,
	}
	lh := &ListHandler[string]{}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 20)
	assert.Equal(t, 0, lh.totalItems)
	assert.Equal(t, 4, lh.totalPages)
	assert.Equal(t, 5, lh.perPage)
}

func TestListUpdateApplyActionWithOffsetNoTotal(t *testing.T) {
	obj := listUpdate[string]{
		totalPages: 4,
		perPage:    5,
	}
	lh := &ListHandler[string]{
		offset: 1,
	}

	obj.applyAction(lh)

	assert.GreaterOrEqual(t, cap(lh.Items), 21)
	assert.Equal(t, 0, lh.totalItems)
	assert.Equal(t, 4, lh.totalPages)
	assert.Equal(t, 5, lh.perPage)
}

// XXX TestListUpdateApplyAction
