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

type mockPageGetter struct {
	mock.Mock
}

func (m *mockPageGetter) GetPage(ctx context.Context, depag *Depaginator[string], req PageRequest) ([]string, error) {
	args := m.Called(ctx, depag, req)

	return To[[]string](args.Get(0)), args.Error(1)
}

func TestPageGetterFuncImplementsPageGetter(t *testing.T) {
	assert.Implements(t, (*PageGetter[string])(nil), PageGetterFunc[string](nil))
}

func TestPageGetterFuncGetPage(t *testing.T) {
	ctx := context.Background()
	depag := &Depaginator[string]{}
	req := PageRequest{}
	pager := &mockPageGetter{}
	pager.On("GetPage", ctx, depag, req).Return([]string{"foo", "bar"}, nil)
	obj := PageGetterFunc[string](pager.GetPage)

	result, err := obj.GetPage(ctx, depag, req)

	assert.NoError(t, err)
	assert.Equal(t, []string{"foo", "bar"}, result)
	pager.AssertExpectations(t)
}

type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) Handle(ctx context.Context, idx int, item string) {
	m.Called(ctx, idx, item)
}

func TestHandlerFuncImplementsHandler(t *testing.T) {
	assert.Implements(t, (*Handler[string])(nil), HandlerFunc[string](nil))
}

func TestHandlerFuncHandle(t *testing.T) {
	ctx := context.Background()
	handler := &mockHandler{}
	handler.On("Handle", ctx, 5, "five")
	obj := HandlerFunc[string](handler.Handle)

	obj.Handle(ctx, 5, "five")

	handler.AssertExpectations(t)
}

type mockStarter struct {
	mock.Mock
}

func (m *mockStarter) Start(ctx context.Context, totalItems, totalPages, perPage int) {
	m.Called(ctx, totalItems, totalPages, perPage)
}

func TestStarterFuncImplementsStarter(t *testing.T) {
	assert.Implements(t, (*Starter)(nil), StarterFunc(nil))
}

func TestStarterFuncStart(t *testing.T) {
	ctx := context.Background()
	starter := &mockStarter{}
	starter.On("Start", ctx, 20, 4, 5)
	obj := StarterFunc(starter.Start)

	obj.Start(ctx, 20, 4, 5)

	starter.AssertExpectations(t)
}

type mockUpdater struct {
	mock.Mock
}

func (m *mockUpdater) Update(ctx context.Context, totalItems, totalPages, perPage int) {
	m.Called(ctx, totalItems, totalPages, perPage)
}

func TestUpdaterFuncImplementsUpdater(t *testing.T) {
	assert.Implements(t, (*Updater)(nil), UpdaterFunc(nil))
}

func TestUpdaterFuncUpdate(t *testing.T) {
	ctx := context.Background()
	updater := &mockUpdater{}
	updater.On("Update", ctx, 20, 4, 5)
	obj := UpdaterFunc(updater.Update)

	obj.Update(ctx, 20, 4, 5)

	updater.AssertExpectations(t)
}

type mockDoner struct {
	mock.Mock
}

func (m *mockDoner) Done(ctx context.Context, totalItems, totalPages, perPage int) {
	m.Called(ctx, totalItems, totalPages, perPage)
}

func TestDonerFuncImplementsDoner(t *testing.T) {
	assert.Implements(t, (*Doner)(nil), DonerFunc(nil))
}

func TestDonerFuncDone(t *testing.T) {
	ctx := context.Background()
	doner := &mockDoner{}
	doner.On("Done", ctx, 20, 4, 5)
	obj := DonerFunc(doner.Done)

	obj.Done(ctx, 20, 4, 5)

	doner.AssertExpectations(t)
}

type mockHandlerFull struct {
	mock.Mock
}

func (m *mockHandlerFull) Handle(ctx context.Context, idx int, item string) {
	m.Called(ctx, idx, item)
}

func (m *mockHandlerFull) Start(ctx context.Context, totalItems, totalPages, perPage int) {
	m.Called(ctx, totalItems, totalPages, perPage)
}

func (m *mockHandlerFull) Update(ctx context.Context, totalItems, totalPages, perPage int) {
	m.Called(ctx, totalItems, totalPages, perPage)
}

func (m *mockHandlerFull) Done(ctx context.Context, totalItems, totalPages, perPage int) {
	m.Called(ctx, totalItems, totalPages, perPage)
}
