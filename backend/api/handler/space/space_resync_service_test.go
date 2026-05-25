/*
 * Copyright 2025 ynet-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package space

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	resyncmodel "github.com/ynet-dev/ynet-studio/backend/api/model/data/space"
)

// fakeResyncInvoker is a hand-rolled stub for the application-layer SVC.
// Captures the request that flowed through and returns canned response/err.
type fakeResyncInvoker struct {
	gotReq *resyncmodel.ResyncESRequest
	called bool
	resp   *resyncmodel.ResyncESResponse
	err    error
}

func (f *fakeResyncInvoker) ResyncES(_ context.Context, req *resyncmodel.ResyncESRequest) (*resyncmodel.ResyncESResponse, error) {
	f.called = true
	f.gotReq = req
	return f.resp, f.err
}

// installFake swaps the package-level invoker getter and returns a restore fn.
func installFake(f *fakeResyncInvoker) func() {
	orig := resyncSvcGetter
	resyncSvcGetter = func() resyncESInvoker { return f }
	return func() { resyncSvcGetter = orig }
}

func newTestServer() *server.Hertz {
	h := server.Default(server.WithDisablePrintRoute(true))
	h.POST("/api/space/resync_es", ResyncES)
	return h
}

// TestResyncES_Handler_BadRequest sends malformed JSON and expects a 400.
// Note: Hertz BindAndValidate only fails on unparseable input here — the
// `required` tag is enforced by thrift codegen at the serialization layer,
// not by Hertz's binder. So we exercise the parse-error path instead.
func TestResyncES_Handler_BadRequest(t *testing.T) {
	fake := &fakeResyncInvoker{}
	restore := installFake(fake)
	defer restore()

	h := newTestServer()

	// Garbage JSON — bind/validate must fail before the SVC is invoked.
	body := []byte(`{"space_id": not-a-number}`)
	w := ut.PerformRequest(h.Engine, "POST", "/api/space/resync_es",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	res := w.Result()

	assert.Equal(t, http.StatusBadRequest, res.StatusCode(), "malformed JSON body must yield 400, got body=%s", string(res.Body()))
	assert.False(t, fake.called, "application SVC must NOT be invoked when bind/validate fails")
}

// TestResyncES_Handler_PropagatesAppError returns a non-nil error from the
// application SVC and verifies the handler maps it to 500 with the message in
// the body.
func TestResyncES_Handler_PropagatesAppError(t *testing.T) {
	fake := &fakeResyncInvoker{
		err: errors.New("es cluster down"),
	}
	restore := installFake(fake)
	defer restore()

	h := newTestServer()

	body := []byte(`{"space_id":"100"}`)
	w := ut.PerformRequest(h.Engine, "POST", "/api/space/resync_es",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	res := w.Result()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode())
	assert.True(t, fake.called, "application SVC must have been invoked")
	require.NotNil(t, fake.gotReq, "gotReq must not be nil")
	assert.Equal(t, int64(100), fake.gotReq.SpaceID)
	assert.True(t, strings.Contains(string(res.Body()), "es cluster down"),
		"500 body should surface the app-layer error message, got %q", string(res.Body()))
}

// TestResyncES_Handler_HappyPath verifies 200 + counts JSON when the app SVC
// returns success.
func TestResyncES_Handler_HappyPath(t *testing.T) {
	fake := &fakeResyncInvoker{
		resp: &resyncmodel.ResyncESResponse{
			Code: 0,
			Msg:  "success",
			Counts: &resyncmodel.ResyncESCounts{
				ProjectDraft:     3,
				CozeResource:     5,
				KbEntries:        2,
				SliceReindexJobs: 7,
			},
		},
	}
	restore := installFake(fake)
	defer restore()

	h := newTestServer()

	body := []byte(`{"space_id":"42"}`)
	w := ut.PerformRequest(h.Engine, "POST", "/api/space/resync_es",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	res := w.Result()

	assert.Equal(t, http.StatusOK, res.StatusCode(), "expected 200 OK, body=%s", string(res.Body()))
	assert.True(t, fake.called)
	require.NotNil(t, fake.gotReq)
	assert.Equal(t, int64(42), fake.gotReq.SpaceID)
	// Body must surface counts.
	bodyStr := string(res.Body())
	assert.Contains(t, bodyStr, `"project_draft":3`)
	assert.Contains(t, bodyStr, `"slice_reindex_jobs":7`)
}

// TestResyncES_Handler_SVCNotInitialized covers the defensive nil guard:
// before InitResyncService runs (or if init failed), the handler must not
// panic — instead return 500.
func TestResyncES_Handler_SVCNotInitialized(t *testing.T) {
	orig := resyncSvcGetter
	resyncSvcGetter = func() resyncESInvoker { return nil }
	defer func() { resyncSvcGetter = orig }()

	h := newTestServer()
	body := []byte(`{"space_id":"1"}`)
	w := ut.PerformRequest(h.Engine, "POST", "/api/space/resync_es",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	res := w.Result()

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode())
	assert.Contains(t, string(res.Body()), "not initialized")
}
