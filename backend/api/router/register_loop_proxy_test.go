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

package router

import (
	"net/http"
	"testing"
)

func TestBuildLoopProxyCookieHeaderKeepsClientSessionKey(t *testing.T) {
	got, injected := buildLoopProxyCookieHeader(
		"i18next=en-US; session_key=studio-session; theme=light",
		"expired-loop-session",
	)

	if injected {
		t.Fatal("expected existing client session_key to be kept without fallback injection")
	}
	want := "i18next=en-US; session_key=studio-session; theme=light"
	if got != want {
		t.Fatalf("cookie header mismatch: got %q, want %q", got, want)
	}
}

func TestBuildLoopProxyCookieHeaderInjectsFallbackWhenSessionMissing(t *testing.T) {
	got, injected := buildLoopProxyCookieHeader("i18next=en-US", "loop-session")

	if !injected {
		t.Fatal("expected fallback session_key to be injected when client session_key is missing")
	}
	want := "i18next=en-US; session_key=loop-session"
	if got != want {
		t.Fatalf("cookie header mismatch: got %q, want %q", got, want)
	}
}

func TestBuildLoopProxyCookieHeaderSkipsEmptyFallback(t *testing.T) {
	got, injected := buildLoopProxyCookieHeader("i18next=en-US", "")

	if injected {
		t.Fatal("expected no injection with empty fallback session")
	}
	want := "i18next=en-US"
	if got != want {
		t.Fatalf("cookie header mismatch: got %q, want %q", got, want)
	}
}

func TestSetLoopInternalIdentityHeadersSignsStudioUser(t *testing.T) {
	headers := http.Header{}
	ok := setLoopInternalIdentityHeaders(headers, loopInternalIdentity{
		UserID:    "402087139",
		SpaceID:   "7652614054615187456",
		Email:     "402087139@qq.com",
		Name:      "Lu",
		Timestamp: 1782864000,
	}, "shared-secret")

	if !ok {
		t.Fatal("expected internal identity headers to be injected")
	}
	if got := headers.Get(loopInternalHeaderUserID); got != "402087139" {
		t.Fatalf("user id header mismatch: got %q", got)
	}
	if got := headers.Get(loopInternalHeaderSpaceID); got != "7652614054615187456" {
		t.Fatalf("space id header mismatch: got %q", got)
	}
	if got := headers.Get(loopInternalHeaderTimestamp); got != "1782864000" {
		t.Fatalf("timestamp header mismatch: got %q", got)
	}
	if got := headers.Get(loopInternalHeaderSignature); got == "" {
		t.Fatal("expected signature header")
	}
}

func TestSetLoopInternalIdentityHeadersSkipsWithoutUserOrSecret(t *testing.T) {
	headers := http.Header{}
	if setLoopInternalIdentityHeaders(headers, loopInternalIdentity{UserID: "42"}, "") {
		t.Fatal("expected no injection without secret")
	}
	if setLoopInternalIdentityHeaders(headers, loopInternalIdentity{}, "shared-secret") {
		t.Fatal("expected no injection without user id")
	}
}

func TestExtractLoopProxySpaceID(t *testing.T) {
	tests := []struct {
		name        string
		queryString string
		headers     http.Header
		body        []byte
		referer     string
		want        string
	}{
		{
			name:        "query workspace id wins",
			queryString: "workspace_id=7652614054615187456",
			body:        []byte(`{"space_id":"111"}`),
			referer:     "http://studio/space/222/bot/333/arrange",
			want:        "7652614054615187456",
		},
		{
			name:        "body space id",
			queryString: "",
			body:        []byte(`{"space_id":7652614054615187457}`),
			referer:     "http://studio/space/222/bot/333/arrange",
			want:        "7652614054615187457",
		},
		{
			name:        "referer space id fallback",
			queryString: "",
			body:        nil,
			referer:     "http://studio/space/7652614054615187458/bot/333/arrange",
			want:        "7652614054615187458",
		},
		{
			name:        "reject non numeric",
			queryString: "workspace_id=abc",
			body:        []byte(`{"space_id":"not-a-number"}`),
			referer:     "http://studio/space/not-a-number/bot/333/arrange",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLoopProxySpaceID(tt.queryString, tt.headers, tt.body, tt.referer)
			if got != tt.want {
				t.Fatalf("space id mismatch: got %q, want %q", got, tt.want)
			}
		})
	}
}
