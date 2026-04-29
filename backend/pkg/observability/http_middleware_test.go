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

package observability

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPMiddleware_IncrementsCounter(t *testing.T) {
	mw := HTTPRequestsMiddleware()

	c := app.NewContext(0)
	c.Request.Header.SetMethod(consts.MethodGet)
	c.Request.SetRequestURI("/api/test")
	c.Response.SetStatusCode(http.StatusOK)

	before := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200"))

	mw(context.Background(), c)

	after := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200"))

	if after != before+1 {
		t.Fatalf("expected counter to increment by 1; before=%v after=%v", before, after)
	}
}

func TestNormalizePath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/api/users/123", "/api/users/:id"},
		{"/api/users/123/posts/456", "/api/users/:id/posts/:id"},
		{"/api/users/abc", "/api/users/abc"},
		{"/api/users/00000000-0000-4000-8000-000000000000", "/api/users/:uuid"},
		{"/", "/"},
		{"", "/"},
	}
	for _, tc := range cases {
		got := normalizePath(tc.in)
		if got != tc.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
