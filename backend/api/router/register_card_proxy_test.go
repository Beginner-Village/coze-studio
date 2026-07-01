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

import "testing"

func TestBuildAOPReadFallbackResponseForRouteSweepEndpoints(t *testing.T) {
	tests := []struct {
		path    string
		bodyKey string
	}{
		{path: "/aop-web/MCP0003.do", bodyKey: "serviceInfoList"},
		{path: "/aop-web/IDC10001.do", bodyKey: "cardList"},
		{path: "/aop-web/IDC10033.do", bodyKey: "cardClassList"},
		{path: "/aop-web/IDC20026.do", bodyKey: "exportTaskLists"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			resp, ok := buildAOPReadFallbackResponse(tt.path)
			if !ok {
				t.Fatalf("expected fallback response for %s", tt.path)
			}

			header, _ := resp["header"].(map[string]any)
			if got := header["errorCode"]; got != "0" {
				t.Fatalf("errorCode mismatch: got %v, want 0", got)
			}

			body, _ := resp["body"].(map[string]any)
			if _, exists := body[tt.bodyKey]; !exists {
				t.Fatalf("expected body key %q in fallback response: %#v", tt.bodyKey, body)
			}
		})
	}
}

func TestBuildAOPReadFallbackResponseSkipsWriteEndpoints(t *testing.T) {
	if _, ok := buildAOPReadFallbackResponse("/aop-web/IDC10002.do"); ok {
		t.Fatal("expected no fallback for card create endpoint")
	}
}
